# devtui — PLAN: Elimina el `app_get_logs` duplicado, mantén el contrato `LogEntry`

> Estado: Borrador para revisión · Objetivo: elimina la tool `app_get_logs` duplicada para que
> la superficie de logs del LLM sea de fuente única en `app`, mientras que devtui mantiene
> la renderización de la TUI interactiva.
>
> ⚠️ Prescriptivo. Ver §5 (Invariantes) y §6 (Aceptación).

---

## 1. Problema (verificado en código)

`app_get_logs` se define dos veces:

- `app/daemon.go:404` (la actual: el daemon la anuncia y la sirve desde el ring del buffer SSE).
- `devtui/mcp.go:78` (`MCPToolName = "app_get_logs"`, `devtui/mcp.go:12`), con
  `mcpGetSectionLogs` (`devtui/mcp.go:124`) y `getSectionLogsPlain` (`devtui/mcp.go:160`).

El daemon corre un `HeadlessTUI` (`app/daemon.go:626`), NO un `DevTUI`, así que la tool de devtui
no es la que llega al LLM a través del daemon. Dos definiciones de un nombre de tool violan
el invariante #3 del maestro (una definición por tool).

Restricción arquitectónica: `app` NO debe importar devtui (`app/handler.go:18`). Por lo tanto
el formateador de logs del LLM no puede vivir en devtui y ser reutilizado por app. El contrato
que vincula los dos procesos es el formato de wire `LogEntry` (`app/sse_publisher.go:15`),
que el cliente SSE de devtui ya consume.

## 2. Objetivo

- devtui deja de definir `app_get_logs` (la propiedad se mueve a app, que es dueña del ring del daemon).
- devtui mantiene su renderización interactiva (`print.go:formatMessage`, `getSectionLogsPlain`)
  para la TUI humana.
- Confirma que los campos `LogEntry` que devtui emite/consume son suficientes para el renderizador de app.

## 3. Diseño

- Elimina la tool MCP `app_get_logs` de `devtui/mcp.go`: borra la entrada de tool `GetMCPTools`,
  `MCPToolName`, `mcpGetSectionLogs`, y el cableado de esquema MCP-only `GetLogsArgs`
  (`devtui/mcp.go:17-94,124-157`). Mantén `getSectionLogsPlain` solo si aún la usa la TUI;
  en otro caso elimínala también.
- devtui sigue publicando logs estructurados a través de la ruta existente así que el
  `PublishTabLog(tabTitle, handlerName, handlerColor, msg)` del daemon (`app/daemon.go:629`)
  siga rellenando `LogEntry`. Sin cambio a `formatMessage` (`print.go:27`) — eso es el
  renderizado de TUI humana.
- La forma de formato plano usada por app (`HH:MM:SS  HANDLER  contenido`) refleja la salida
  de `formatMessage(styled=false)` de devtui (`print.go:59`) así que ambas se ven consistentes;
  esta es una convención documentada, no una función compartida (regla de importación).

## 4. Pasos

1. Borra la tool `app_get_logs` + helpers de `devtui/mcp.go`.
2. Si `getSectionLogsPlain` queda sin usar, elimínala; en otro caso déjala para la TUI.
3. Confirma que `DevTUI` siga satisfaciendo cualquier contrato `mcp.ToolProvider`/`GetMCPTools`
   que el cableado del cliente espera (puede ahora devolver un conjunto vacío de tools).

## 5. Invariantes / prohibiciones

- **No** cambies la forma JSON de `LogEntry` (`app/sse_publisher.go:15`) — es el contrato
  app↔devtui.
- **No** hagas que app importe devtui para "compartir" un formateador.
- **No** alteres la renderización de TUI humana (`formatMessage`, timestamps, colores de handler).

## 6. Aceptación

- `app_get_logs` se define en exactamente un lugar (`app`).
- La TUI interactiva sigue renderizando logs idénticamente (tiempo + handler + contenido coloreado).
- `go build ./... && go test ./...` en verde para devtui.

## 7. Tests

- Ajusta/mantén tests de devtui así que la eliminación de la tool MCP no rompa los tests de
  renderización de la TUI.
- Sin nuevo test MCP en devtui (la tool ya no vive aquí; su cobertura se mueve a app §5).
