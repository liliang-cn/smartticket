package mcp

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/company/smartticket/internal/branding"
	"github.com/company/smartticket/internal/models"
)

// brandingView is the schema-safe MCP view of the white-label configuration.
type brandingView struct {
	AppName       string `json:"app_name" jsonschema:"product name shown in the console and login"`
	AppSubtitle   string `json:"app_subtitle,omitempty" jsonschema:"small label under the app name"`
	WorkspaceName string `json:"workspace_name,omitempty" jsonschema:"workspace label in the top bar"`
	PrimaryColor  string `json:"primary_color,omitempty" jsonschema:"accent color as a hex string"`
	LoginTagline  string `json:"login_tagline,omitempty" jsonschema:"headline on the sign-in page"`
	LoginSubtext  string `json:"login_subtext,omitempty" jsonschema:"subtext on the sign-in page"`
	HasLogo       bool   `json:"has_logo" jsonschema:"whether a custom logo image is set"`
}

func brandingViewFrom(b *models.Branding) brandingView {
	if b == nil {
		return brandingView{}
	}
	return brandingView{
		AppName: b.AppName, AppSubtitle: b.AppSubtitle, WorkspaceName: b.WorkspaceName,
		PrimaryColor: b.PrimaryColor, LoginTagline: b.LoginTagline, LoginSubtext: b.LoginSubtext,
		HasLogo: b.LogoPath != "",
	}
}

// registerBrandingTools registers the branding/settings MCP tools. Reading is
// open to any authenticated session; writes require settings:write.
func registerBrandingTools(s *mcp.Server, b Backend) {
	registerTool(s, "branding_get",
		"Get the deployment's white-label branding (names, accent color, login text).",
		"",
		func(ctx context.Context, _ struct{}) (brandingView, string, error) {
			return brandingGet(ctx, b)
		})

	registerTool(s, "branding_update",
		"Update the deployment's branding. Only provided fields are changed.",
		"settings:write",
		func(ctx context.Context, in brandingUpdateInput) (brandingView, string, error) {
			return brandingUpdate(ctx, b, in)
		})

	registerTool(s, "branding_delete_logo",
		"Remove the custom logo image, reverting to the default glyph.",
		"settings:write",
		func(ctx context.Context, _ struct{}) (brandingView, string, error) {
			return brandingDeleteLogo(ctx, b)
		})
}

// ---- schemas ----

type brandingUpdateInput struct {
	AppName       *string `json:"app_name,omitempty" jsonschema:"new product name"`
	AppSubtitle   *string `json:"app_subtitle,omitempty" jsonschema:"new subtitle"`
	WorkspaceName *string `json:"workspace_name,omitempty" jsonschema:"new workspace label"`
	PrimaryColor  *string `json:"primary_color,omitempty" jsonschema:"new accent color as a hex string, e.g. #f59e0b"`
	LoginTagline  *string `json:"login_tagline,omitempty" jsonschema:"new sign-in headline"`
	LoginSubtext  *string `json:"login_subtext,omitempty" jsonschema:"new sign-in subtext"`
}

// ---- closures ----

func brandingGet(_ context.Context, b Backend) (brandingView, string, error) {
	r, err := b.GetBranding()
	if err != nil {
		return brandingView{}, "", err
	}
	return brandingViewFrom(r), fmt.Sprintf("Branding: %q (accent %s).", r.AppName, r.PrimaryColor), nil
}

func brandingUpdate(_ context.Context, b Backend, in brandingUpdateInput) (brandingView, string, error) {
	req := &branding.UpdateRequest{
		AppName: in.AppName, AppSubtitle: in.AppSubtitle, WorkspaceName: in.WorkspaceName,
		PrimaryColor: in.PrimaryColor, LoginTagline: in.LoginTagline, LoginSubtext: in.LoginSubtext,
	}
	r, err := b.UpdateBranding(req)
	if err != nil {
		return brandingView{}, "", err
	}
	return brandingViewFrom(r), "Branding updated.", nil
}

func brandingDeleteLogo(_ context.Context, b Backend) (brandingView, string, error) {
	r, err := b.DeleteBrandingLogo()
	if err != nil {
		return brandingView{}, "", err
	}
	return brandingViewFrom(r), "Logo removed.", nil
}
