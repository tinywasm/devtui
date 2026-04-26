# PLAN: tinywasm/devtui — Señalización de cierre en modo standalone

## Problema

En modo standalone (no clientMode), cuando el usuario cierra el TUI (Ctrl+C),
el contenido del TUI permanece visible en el terminal después de salir. El proceso
queda bloqueado y el shell lo termina con SIGKILL, que no permite ejecutar
`ExitAltScreen` → el buffer alternativo no se restaura.

### Cadena de eventos actual (rota)

```
usuario Ctrl+C
  → devtui.Update: shutdownMsg → ClearScreen + ExitAltScreen + Quit  ← correcto
  → tea.Run() retorna  ← correcto
  → sseWg.Wait() → devtui.Start() llama wg.Done()  ← correcto
  → wg.Wait() en start.go BLOQUEA  ← ❌ ExitChan nunca se cierra
      (HTTP server goroutine espera <-ExitChan indefinidamente)
  → shell envía SIGKILL al proceso bloqueado
  → terminal no recibe ExitAltScreen → contenido queda visible  ← síntoma
```

### Por qué el clientMode no tiene este problema

En clientMode (`bootstrap.go:runClient`), quien cierra `ExitChan` es `runClient`
explícitamente con `close(exitChan)` **después** de que `Start()` retorna.
En standalone, `Start()` nunca retorna porque nadie cierra `ExitChan`.

## Solución

`devtui.Start()` ya recibe un `*sync.WaitGroup` opcional. Extender para recibir
también un canal de exit opcional (`chan bool` o `chan struct{}`). Cuando el TUI
termina limpiamente (después de `tea.Run()` y `sseWg.Wait()`), cerrar ese canal
para que los goroutines dependientes (`<-ExitChan`) puedan terminar.

### API — `devtui.Start()`

No cambiar la firma pública. Usar el patrón variadic `...any` ya existente:

```go
// devtui/init.go — Start()
//
// Parameters:
//   - args ...any: Optional arguments. Supported types:
//   - *sync.WaitGroup: called Done() when the TUI exits.
//   - chan bool:       closed when the TUI exits cleanly, so goroutines
//                     blocked on <-exitChan can terminate.
func (h *DevTUI) Start(args ...any) {
    var wg *sync.WaitGroup
    var exitChan chan bool

    for _, arg := range args {
        switch v := arg.(type) {
        case *sync.WaitGroup:
            wg = v
        case chan bool:
            exitChan = v
        }
    }
    if wg != nil {
        defer wg.Done()
    }

    // ... tea.Run() + sseWg.Wait() ...

    // Al terminar limpiamente, cerrar el canal para liberar goroutines dependientes
    if exitChan != nil {
        select {
        case <-exitChan: // ya cerrado (clientMode lo cierra externamente)
        default:
            close(exitChan)
        }
    }
}
```

## Archivos afectados

| Archivo | Cambio |
|---------|--------|
| `devtui/init.go` | `Start()`: extraer `chan bool` de args y cerrarlo al salir |
| `devtui/shutdown_test.go` | Test de regresión: `Start()` retorna y cierra `exitChan` |

## Paso 2 — Test de regresión del cierre standalone

### Por qué el test funciona sin TTY

En entornos de test (sin TTY real), `tea.Run()` falla inmediatamente con un error
y retorna. Esto provoca que `Start()` continúe al bloque `sseWg.Wait()` y regrese.
Con la corrección aplicada, `exitChan` se cierra antes de que `Start()` retorne.
No se necesita un TTY real ni llamar `Shutdown()` explícitamente.

### Implementación — `devtui/shutdown_test.go`

```go
package devtui_test

import (
    "sync"
    "testing"
    "time"
)

// TestStartClosesExitChan verifies that Start() closes the exitChan when it returns,
// even when tea.Run() fails immediately (no TTY in test environment).
func TestStartClosesExitChan(t *testing.T) {
    tui := DefaultTUIForTest(t)

    var wg sync.WaitGroup
    exitChan := make(chan bool)

    wg.Add(1)
    go tui.Start(&wg, exitChan)

    select {
    case <-exitChan:
        // exitChan closed — Start() returned cleanly
    case <-time.After(3 * time.Second):
        t.Fatal("Start() did not close exitChan within 3s")
    }

    // Also verify the WaitGroup is released
    done := make(chan struct{})
    go func() { wg.Wait(); close(done) }()
    select {
    case <-done:
    case <-time.After(time.Second):
        t.Fatal("WaitGroup not released after exitChan closed")
    }
}
```

### Notas

- `DefaultTUIForTest` ya existe en `handler_test.go` — crea un `DevTUI` con `SetTestMode(true)`.
- No hay `t.Parallel()` porque `tea.NewProgram` puede acceder a `os.Stdout` globalmente.
- El timeout de 3s es holgado: en CI el fallo es inmediato (<50ms).

## Verificación

```bash
go test ./...                                   # todos los tests deben pasar
go test -run TestStartClosesExitChan -v ./...   # test específico
```
