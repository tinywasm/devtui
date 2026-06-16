# devtui — MCP Integration

## Decisión de diseño

devtui satisface la interfaz MCP cliente (`GetMCPTools`, `DispatchAction`, `GetHandlerStates`,
`Name`, `SetLog`) pero **no expone tools MCP propias**

- `GetMCPTools()` devuelve `nil` — devtui no posee ninguna tool.
- `DispatchAction` devuelve `false` — las acciones se reenvían al daemon, no se despachan localmente.
- `GetHandlerStates` devuelve `nil` — devtui es cliente, no servidor de estado.

## Por qué `app_get_logs` no vive aquí

La tool `app_get_logs` es propiedad de `app` (`app/daemon.go`), que es dueña del ring buffer
SSE (`app/sse_publisher.go:31`). El contrato app↔devtui es el formato wire `LogEntry`
(`app/sse_publisher.go:15`); devtui lo consume para renderizar la TUI interactiva, pero no lo
re-expone al LLM.

Restricción arquitectónica: `app` no debe importar `devtui` (`app/handler.go:18`). Por tanto
el formateador de logs del LLM vive en `app`.

## Renderización TUI (sin cambio)

`formatMessage` (`print.go:27`) y `getSectionLogsPlain` siguen siendo la superficie de
renderizado para el usuario humano. No se comparten con `app` (regla de importación).

El formato plano `HH:MM:SS  HANDLER  contenido` que `app` usa para el LLM refleja la salida
de `formatMessage(styled=false)` — convención documentada, no función compartida.
