package gtm

import (
	"context"
	"fmt"

	"gtm-mcp-server/auth"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// RegisterTools adds all GTM tools to the MCP server.
func RegisterTools(server *mcp.Server) {
	// Read operations
	registerListAccounts(server)
	registerUpdateAccount(server)
	registerListContainers(server)
	registerListWorkspaces(server)
	registerListTags(server)
	registerGetTag(server)
	registerListTriggers(server)
	registerGetTrigger(server)
	registerListVariables(server)
	registerGetVariable(server)
	registerListFolders(server)
	registerGetFolderEntities(server)
	registerListTemplates(server)
	registerGetTemplate(server)
	registerListVersions(server)

	// Write operations
	registerCreateTag(server)
	registerUpdateTag(server)
	registerDeleteTag(server)
	registerCreateTrigger(server)
	registerUpdateTrigger(server)
	registerDeleteTrigger(server)
	registerCreateVariable(server)
	registerUpdateVariable(server)
	registerDeleteVariable(server)
	registerCreateContainer(server)
	registerUpdateContainer(server)
	registerDeleteContainer(server)
	registerCreateWorkspace(server)

	// Workspace status
	registerGetWorkspaceStatus(server)

	// Version operations
	registerCreateVersion(server)
	registerPublishVersion(server)

	// Template operations
	registerImportGalleryTemplate(server)
	registerCreateTemplate(server)
	registerUpdateTemplate(server)
	registerDeleteTemplate(server)

	// Built-in variables
	registerListBuiltInVariables(server)
	registerEnableBuiltInVariables(server)
	registerDisableBuiltInVariables(server)

	// Clients (server-side containers)
	registerListClients(server)
	registerGetClient(server)
	registerCreateClient(server)
	registerUpdateClient(server)
	registerDeleteClient(server)

	// Transformations (server-side containers)
	registerListTransformations(server)
	registerGetTransformation(server)
	registerCreateTransformation(server)
	registerUpdateTransformation(server)
	registerDeleteTransformation(server)

	// Templates (help LLMs with correct parameter formats)
	registerGetTagTemplates(server)
	registerGetTriggerTemplates(server)

	// Resources (URI-based read access)
	RegisterResources(server)

	// Prompts (template workflows)
	RegisterPrompts(server)
}

// getClient creates a GTM client using the service account token source from context.
func getClient(ctx context.Context) (*Client, error) {
	saTS := auth.GetSATokenSource(ctx)
	if saTS == nil {
		return nil, fmt.Errorf("not authenticated - service account not available in context")
	}
	return NewClient(ctx, saTS)
}
