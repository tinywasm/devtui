# PLAN: Limpieza de Tests en devtui

## Diagnóstico

### ✅ Data Race — RESUELTO
`TestSSEClient_Reconnection` en [client_mode_test.go](../client_mode_test.go) tenía un data race: `connectionCount int` compartido entre el handler HTTP (varias goroutines del servidor httptest) y el loop del test, sin sincronización. Corregido usando `atomic.Int32`.

### Ruido excesivo de logs
**Los tests pasan** (`ok github.com/tinywasm/devtui`). El problema restante es **ruido**: 127 llamadas a `t.Log`/`t.Logf` en 17 archivos generan salida verbosa incluso en ejecuciones exitosas.

En Go, `t.Log` y `t.Logf` se muestran siempre con `-v` y **también** cuando el test falla. El estándar correcto: solo loguear cuando el test falla usando `t.Errorf` para dar contexto, o usar `t.Log` únicamente dentro de bloques `if got != want`.

---

## Archivos con más ruido (ordenados por impacto)

| Archivo | `t.Log` informativos |
|---|---|
| `color_conflict_test.go` | 19 |
| `ui_display_bug_test.go` | 17 |
| `real_user_scenario_test.go` | 17 |
| `chat_handler_test.go` | 17 |
| `layout_test.go` | 11 |
| `manual_scenario_test.go` | 10 |
| `handler_value_update_test.go` | 5 |
| `user_scenario_test.go` | 4 |
| `userKeyboard_test.go` | 4 |
| `space_key_test.go` | 4 |
| `refresh_tab_test.go` | 4 |
| `field_editing_bug_test.go` | 4 |

---

## Regla a aplicar

```go
// ❌ ANTES — siempre imprime aunque el test pase
t.Logf("✅ PASS: '%s' correctly detected as %v", tc.content, detectedType)
t.Log("State 1: DevTUI selects field -> handler shows content")
t.Log("CONCLUSION: DetectMessageType works correctly")

// ✅ DESPUÉS — solo imprime si falla
if got != want {
    t.Errorf("DetectMessageType(%q) = %v, want %v", input, got, want)
}
```

Los `t.Log` de contexto de fallo (dentro de un `if` de error) son válidos y se mantienen.

---

## Etapas

### Etapa 1 — `color_conflict_test.go` (19 logs)
- Eliminar encabezados decorativos: `TESTING CENTRALIZED...`, `=====`, `CONCLUSION:`, `SOLUTION:`, `RESULT:`, `BENEFIT:`, `CONSISTENCY:`
- Convertir `t.Logf("✅ PASS: ...")` → eliminar (el PASS ya lo reporta el framework)
- Mantener solo los `t.Errorf` de fallo real

### Etapa 2 — `chat_handler_test.go` (17 logs)
- Eliminar todos los `t.Logf("State N: ...")`, `t.Logf("Phase N: ...")`, `=== TESTING ...`, `=== COMPLETED ===`
- Los estados del chat son de diseño, no de verificación — pertenecen a comentarios del código, no a logs de test

### Etapa 3 — `ui_display_bug_test.go` y `real_user_scenario_test.go` (17 c/u)
- Mismo patrón: logs de "steps" y "phases" que describen el flujo sin verificar nada
- Reemplazar por assertions directas con `t.Errorf`

### Etapa 4 — `layout_test.go`, `manual_scenario_test.go` (11-10 logs)
- Eliminar logs de progreso de pasos
- Conservar solo contexto en `t.Errorf` cuando sea necesario

### Etapa 5 — Archivos menores (4-5 logs c/u)
- `handler_value_update_test.go`, `user_scenario_test.go`, `userKeyboard_test.go`, `space_key_test.go`, `refresh_tab_test.go`, `field_editing_bug_test.go`
- Limpieza rápida: eliminar logs de step/initial state informativos

---

## Criterio de aceptación

```bash
go test ./... 2>&1
# Salida esperada:
ok  github.com/tinywasm/devtui  X.XXXs
```

Sin líneas de `t.Log` en la salida de un run exitoso. Con `-v` solo deben aparecer los nombres de tests (`=== RUN`, `--- PASS`), sin mensajes de estado internos.

```bash
# Verificar que no queda ruido informativo
go test ./... -v 2>&1 | grep -v "=== RUN\|--- PASS\|--- FAIL\|^ok\|^\?" | grep -v "^$"
# Resultado esperado: vacío o solo mensajes de error reales
```

---

## Lo que NO cambiar

- `t.Errorf` / `t.Fatalf` — estos son correctos
- `t.Log` dentro de bloques `if err != nil` o `if got != want` — contexto útil en fallo
- `t.Log` en helpers de test compartidos que se llaman desde múltiples tests
