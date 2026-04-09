# PLAN: Move SHORTCUTS Tab to Last Position

## Objetivo

Mover la creación del tab SHORTCUTS del momento de construcción (`NewTUI`) al momento de arranque (`Start`), para que siempre aparezca al final de todos los tabs registrados por el usuario.

## Problema Actual

En `init.go:127`, `createShortcutsTab(tui)` se llama dentro de `NewTUI()`, antes de que el usuario agregue sus propios tabs. Resultado: SHORTCUTS ocupa `TabSections[0]` y es lo primero que ve el usuario al arrancar la TUI en `tinywasm/app`.

```
NewTUI()
  └── createShortcutsTab()   ← posición 0
  
NewTabSection("BUILD")       ← posición 1
NewTabSection("TEST")        ← posición 2
...
Start()
```

## Solución

Mover `createShortcutsTab(tui)` a `Start()`, justo antes de `h.tea.Run()`.

```
NewTUI()                     ← sin shortcuts

NewTabSection("BUILD")       ← posición 0
NewTabSection("TEST")        ← posición 1
...
Start()
  └── createShortcutsTab()   ← posición N (último)
  └── h.tea.Run()
```

Si no se agrega ningún tab de usuario, SHORTCUTS queda en posición 0 (único tab).

## Breaking Changes

| Área | Antes | Después |
|------|-------|---------|
| `TabSections[0]` | SHORTCUTS | Primer tab de usuario |
| `activeTab = 0` al arrancar | Muestra SHORTCUTS | Muestra primer tab de usuario ✓ |
| Tests que llaman solo `NewTUI()` | `len(TabSections) == 1` (SHORTCUTS) | `len(TabSections) == 0` |

## Archivos a Modificar

### 1. `init.go`

**Remover** de `NewTUI()` (línea 127):
```go
// Always add SHORTCUTS tab first   ← eliminar comentario
createShortcutsTab(tui)             ← eliminar llamada
```

**Agregar** en `Start()` antes de `h.tea.Run()`:
```go
// Add SHORTCUTS tab last, after all user tabs are registered
createShortcutsTab(h)
```

### 2. `integration_test.go`

Línea 55: `tui.TabSections[0]` actualmente accede a SHORTCUTS (incorrecto semánticamente). Después del cambio accederá a "Datos personales" (correcto). **No requiere cambio de código**, pero el comportamiento implícito mejora.

### 3. Tests que asuman `len(TabSections) >= 1` post-`NewTUI()`

Verificar que ningún test dependa de que SHORTCUTS exista antes de llamar `Start()`. Actualmente ninguno lo hace explícitamente.

## Pasos de Ejecución

- [ ] **Step 1** — Remover `createShortcutsTab(tui)` de `NewTUI()` en `init.go:127`
- [ ] **Step 2** — Agregar `createShortcutsTab(h)` en `Start()` en `init.go`, antes de `h.tea.Run()`
- [ ] **Step 3** — Correr tests: `go test ./...`


## Notas

- `HeadlessTUI` (interfaz `TuiInterface`) no implementa `createShortcutsTab`, no hay impacto.
- `testMode` no llama `Start()`, por lo que SHORTCUTS no se agregará en tests — esto es correcto ya que los tests no necesitan ese tab.
- El índice `activeTab` permanece en `0`, lo que ahora apunta al primer tab de usuario (comportamiento deseado).
