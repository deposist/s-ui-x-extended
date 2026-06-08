package importxui

import "encoding/json"

func buildOutJson(inType string, tag string, server string, port int, tlsBlock map[string]any, transport map[string]any, flow string) (json.RawMessage, error) {
	out := map[string]any{
		"type":        inType,
		"tag":         tag,
		"server":      server,
		"server_port": port,
	}
	if tlsBlock != nil {
		out["tls"] = tlsBlock
	}
	if transport != nil {
		out["transport"] = transport
	}
	if flow != "" {
		out["flow"] = flow
	}
	return marshalJSON(out)
}

func buildAddrs() json.RawMessage {
	return json.RawMessage(`[]`)
}
