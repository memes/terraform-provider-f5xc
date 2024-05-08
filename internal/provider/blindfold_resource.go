package provider

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/memes/f5xc"
	"github.com/memes/f5xc/blindfold"
)

var (
	_ resource.Resource              = &blindfoldResource{}
	_ resource.ResourceWithConfigure = &blindfoldResource{}
)

type blindfoldResource struct {
	client  *http.Client
	timeout time.Duration
}

type policyDocumentModel struct {
	Name      types.String `tfsdk:"name"`
	Namespace types.String `tfsdk:"namespace"`
}

type blindfoldResourceModel struct {
	ID             types.String        `tfsdk:"id"`
	Sealed         types.String        `tfsdk:"sealed"`
	Plaintext      types.String        `tfsdk:"plaintext"`
	PolicyDocument policyDocumentModel `tfsdk:"policy_document"`
	Vesctl         types.String        `tfsdk:"vesctl"`
}

func NewBlindfoldResource() resource.Resource {
	return &blindfoldResource{}
}

// Implement the Metadata function for Resource interface.
func (r *blindfoldResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_blindfold"
}

// Implement the Schema function for Resource interface. Blindfold resources are configured to accept plaintext data,
// which will be stored as part of the resource's state unfortunately, and a name+namespace reference to a secret
// policy document. A definitive path to vesctl can be provided as an option.
func (r *blindfoldResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Generates a blindfolded secret from a base64 encoded source string.\n\n" +
			"NOTE: The Terraform state *will include the unencrypted source value* that was provided " +
			"through the `plaintext` attribute.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The computed resource identifier for the blindfolded secret.",
				Computed:    true,
			},
			"sealed": schema.StringAttribute{
				Description: "The base64 encoded, sealed data resulting from a blindfold.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"plaintext": schema.StringAttribute{
				Description: "The base64 encoded plaintext data that will be blindfolded.",
				Required:    true,
				Sensitive:   true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"policy_document": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						Description: "The name of the F5XC PolicyDocument to use for blindfold.",
						Required:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
					"namespace": schema.StringAttribute{
						Description: "The namespace of the F5XC PolicyDocument to use for blindfold.",
						Required:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
				},
			},
			"vesctl": schema.StringAttribute{
				MarkdownDescription: "The path to `vesctl` binary to use for blindfolding. If " +
					"unspecified, the first vesctl binary found in PATH will be used",
				Optional: true,
			},
		},
	}
}

// Implement the Configure function for Resource interface.
func (r *blindfoldResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	cfg, ok := req.ProviderData.(*f5XCConfig)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *f5XCConfig, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	r.client = cfg.client
	r.timeout = cfg.timeout
}

// Implement the Create function for Resource interface. Blindfold resources are entirely ephemeral and any change in
// state that triggers the Create function will return a newly blindfolded secret value.
func (r *blindfoldResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { //nolint:gocritic // Provider interface passes CreateRequest by value.
	tflog.Info(ctx, "Creating blindfold resource")
	var model blindfoldResourceModel
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx = tflog.SetField(ctx, "policy_doc_name", model.PolicyDocument.Name.ValueString())
	ctx = tflog.SetField(ctx, "policy_doc_namespace", model.PolicyDocument.Namespace.ValueString())
	ctx = tflog.SetField(ctx, "vesctl", model.Vesctl.ValueString())

	id, err := uuid.NewRandom()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error computing id",
			"Failed to compute a new id for the resource, unexpected error: "+err.Error(),
		)
	}
	model.ID = types.StringValue(id.String())

	tflog.Debug(ctx, "Decoding plaintext value from base64")
	plaintext, err := base64.StdEncoding.DecodeString(model.Plaintext.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error decoding Base64 plaintext",
			"Failed to decode base64 plaintext to byte array, unexpected error: "+err.Error(),
		)
	}

	tflog.Debug(ctx, "Fetching Public Key")
	clientCtx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	pubKey, err := f5xc.GetPublicKey(clientCtx, r.client, nil)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error retrieving PublicKey",
			"Could not retrieve PublicKey, unexpected error: "+err.Error(),
		)
		return
	}
	if pubKey == nil {
		resp.Diagnostics.AddError(
			"Error retrieving PublicKey",
			"PublicKey was not found for this account",
		)
		return
	}
	cancel()

	tflog.Debug(ctx, "Fetching Secret Policy Document")
	clientCtx, cancel = context.WithTimeout(ctx, r.timeout)
	defer cancel()
	policyDoc, err := f5xc.GetSecretPolicyDocument(clientCtx, r.client, model.PolicyDocument.Name.ValueString(), model.PolicyDocument.Namespace.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error retrieving SecretPolicyDocument",
			"Could not retrieve SecretPolicyDocument, unexpected error: "+err.Error(),
		)
		return
	}
	if policyDoc == nil {
		resp.Diagnostics.AddError(
			"Error retrieving SecretPolicyDocument",
			"SecretPolicyDocument was not found; check the assigned values for name and namespace",
		)
		return
	}
	cancel()

	tflog.Debug(ctx, "Executing blindfold")
	clientCtx, cancel = context.WithTimeout(ctx, r.timeout)
	defer cancel()
	sealed, err := blindfold.Seal(clientCtx, model.Vesctl.ValueString(), plaintext, pubKey, policyDoc)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error blindfolding data",
			"Failed to blindfold data, unexpected error: "+err.Error(),
		)
		return
	}
	model.Sealed = types.StringValue(string(sealed))

	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Implement the Read function for Resource interface. Blindfold resources do not create any state to read in, so this
// function does nothing. Terraform state will be unchanged.
func (r *blindfoldResource) Read(_ context.Context, _ resource.ReadRequest, _ *resource.ReadResponse) { //nolint:gocritic // Provider interface passes ReadRequest by value.
}

// Implement the Update function for Resource interface. Blindfold resources do not create any state to update, so this
// function sets post-update state to the same values as present in the prior plan.
func (r *blindfoldResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) { //nolint:gocritic // Provider interface passes UpdateRequest by value.
	tflog.Info(ctx, "Updating blindfold resource")
	var model blindfoldResourceModel
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Implement the Delete function for Resource interface. Blindfold resources do not create any state to clean up, so this
// function does nothing. Terraform state will be deleted as long as the function does not add diagnostics to the response.
func (r *blindfoldResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) { //nolint:gocritic // Provider interface passes DeleteRequest by value.
}
