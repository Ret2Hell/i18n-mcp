package mcpserver

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/Ret2Hell/i18n-mcp/internal/app"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func registerPatchResources(server *mcp.Server, a *app.App) {
	handler := func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		uri, err := url.Parse(req.Params.URI)
		if err != nil || uri.Scheme != "i18n" || uri.Host != "patches" || uri.RawQuery != "" || uri.Fragment != "" || len(req.Params.URI) > 2048 {
			return nil, mcp.ResourceNotFoundError(req.Params.URI)
		}
		parts := strings.Split(strings.TrimPrefix(uri.Path, "/"), "/")
		if len(parts) < 3 {
			return nil, mcp.ResourceNotFoundError(req.Params.URI)
		}
		kind := parts[1]
		index := 0
		offset := 0
		if kind == "manifest" && len(parts) == 3 {
			offset, err = strconv.Atoi(parts[2])
		} else if kind == "file" && len(parts) == 4 {
			index, err = strconv.Atoi(parts[2])
			if err == nil {
				offset, err = strconv.Atoi(parts[3])
			}
		} else {
			return nil, mcp.ResourceNotFoundError(req.Params.URI)
		}
		if err != nil {
			return nil, fmt.Errorf("invalid patch resource index/offset")
		}
		page, err := a.Previews.Page(ctx, parts[0], kind, index, offset)
		if err != nil {
			return nil, err
		}
		return jsonResource(req.Params.URI, page)
	}
	for _, uri := range []string{"i18n://patches/{snapshot}/manifest/{offset}", "i18n://patches/{snapshot}/file/{index}/{offset}"} {
		server.AddResourceTemplate(&mcp.ResourceTemplate{URITemplate: uri, Name: "patch_" + strings.Split(uri, "/")[4], Title: "Immutable reviewed patch page", MIMEType: "application/json", Description: "Caller-scoped patch snapshot; file pages contain byte-exact base64 chunks with a complete-patch SHA-256 and nextResource."}, handler)
	}
}
