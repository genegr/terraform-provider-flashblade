package provider

import (
	"context"
	"fmt"
	
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"terraform-provider-flashblade/internal/client"
	fb "terraform-provider-flashblade/fb_sdk"
)

var (
	_ resource.Resource                = &fileSystemResource{}
	_ resource.ResourceWithConfigure   = &fileSystemResource{}
	_ resource.ResourceWithImportState = &fileSystemResource{}
)

// Define the attribute types for our nested objects.
var nfsAttributeTypes = map[string]attr.Type{
	"v3_enabled":   types.BoolType,
	"v4_1_enabled": types.BoolType,
	"rules":        types.StringType,
}

var smbAttributeTypes = map[string]attr.Type{
	"enabled":                         types.BoolType,
	"continuous_availability_enabled": types.BoolType,
	"client_policy_name":              types.StringType,
	"share_policy_name":               types.StringType,
}

var multiProtocolAttributeTypes = map[string]attr.Type{
	"access_control_style": types.StringType,
	"safeguard_acls":       types.BoolType,
}

func NewFileSystemResource() resource.Resource {
	return &fileSystemResource{}
}

type fileSystemResource struct {
	client *client.Client
}

// --- MODELS ---
type fileSystemResourceModel struct {
	ID                         types.String `tfsdk:"id"`
	Name                       types.String `tfsdk:"name"`
	Provisioned                types.Int64  `tfsdk:"provisioned"`
	HardLimitEnabled           types.Bool   `tfsdk:"hard_limit_enabled"`
	DefaultGroupQuota          types.Int64  `tfsdk:"default_group_quota"`
	DefaultUserQuota           types.Int64  `tfsdk:"default_user_quota"`
	SnapshotDirectoryEnabled   types.Bool   `tfsdk:"snapshot_directory_enabled"`
	Writable                   types.Bool   `tfsdk:"writable"`
	RequestedPromotionState    types.String `tfsdk:"requested_promotion_state"`
	QosPolicyName              types.String `tfsdk:"qos_policy_name"`
	Created                    types.Int64  `tfsdk:"created"`
	Destroyed                  types.Bool   `tfsdk:"destroyed"`
	TimeRemaining              types.Int64  `tfsdk:"time_remaining"`
	Nfs                        types.Object `tfsdk:"nfs"`
	Smb                        types.Object `tfsdk:"smb"`
	MultiProtocol              types.Object `tfsdk:"multi_protocol"`
}

type nfsModel struct {
	V3Enabled  types.Bool   `tfsdk:"v3_enabled"`
	V41Enabled types.Bool   `tfsdk:"v4_1_enabled"`
	Rules      types.String `tfsdk:"rules"`
}

type smbModel struct {
	Enabled                       types.Bool   `tfsdk:"enabled"`
	ContinuousAvailabilityEnabled types.Bool   `tfsdk:"continuous_availability_enabled"`
	ClientPolicyName              types.String `tfsdk:"client_policy_name"`
	SharePolicyName               types.String `tfsdk:"share_policy_name"`
}

type multiProtocolModel struct {
	AccessControlStyle types.String `tfsdk:"access_control_style"`
	SafeguardAcls      types.Bool   `tfsdk:"safeguard_acls"`
}

func (r *fileSystemResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_file_system"
}

// --- SCHEMA ---
func (r *fileSystemResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Pure Storage FlashBlade file system.",
		Attributes: map[string]schema.Attribute{
			"id":          schema.StringAttribute{Description: "A non-modifiable, globally unique ID chosen by the system.", Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"name":        schema.StringAttribute{Description: "The name of the file system.", Required: true},
			"provisioned": schema.Int64Attribute{Description: "The provisioned size of the file system in bytes.", Optional: true, Computed: true},
			"hard_limit_enabled": schema.BoolAttribute{
				Description:   "If set to true, the file system's size is used as a hard limit quota.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"default_group_quota": schema.Int64Attribute{
				Description:   "The default space quota for a group writing to this file system.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"default_user_quota": schema.Int64Attribute{
				Description:   "The default space quota for a user writing to this file system.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"snapshot_directory_enabled": schema.BoolAttribute{
				Description:   "If true, a hidden .snapshot directory is present in each directory of the file system.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"writable": schema.BoolAttribute{
				Description:   "Whether the file system is writable or not.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"requested_promotion_state": schema.StringAttribute{
				Description:   "The requested promotion state of the file system. Can be `promoted` or `demoted`.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"qos_policy_name": schema.StringAttribute{Description: "The name of the Quality of Service policy for the file system.", Optional: true, Computed: true},
			"created":         schema.Int64Attribute{Description: "Creation timestamp of the file system.", Computed: true},
			"destroyed":       schema.BoolAttribute{Description: "Is the file system destroyed?", Computed: true},
			"time_remaining":  schema.Int64Attribute{Description: "Time in milliseconds before the file system is eradicated.", Computed: true},
			"nfs": schema.SingleNestedAttribute{
				Description: "NFS protocol configuration.",
				Optional:    true,
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"v3_enabled":   schema.BoolAttribute{Optional: true, Computed: true},
					"v4_1_enabled": schema.BoolAttribute{Optional: true, Computed: true},
					"rules":        schema.StringAttribute{Optional: true, Computed: true},
				},
			},
			"smb": schema.SingleNestedAttribute{
				Description: "SMB protocol configuration.",
				Optional:    true,
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"enabled":                         schema.BoolAttribute{Optional: true, Computed: true},
					"continuous_availability_enabled": schema.BoolAttribute{Optional: true, Computed: true},
					"client_policy_name":              schema.StringAttribute{Description: "The name of the SMB client policy.", Optional: true, Computed: true},
					"share_policy_name":               schema.StringAttribute{Description: "The name of the SMB share policy.", Optional: true, Computed: true},
				},
			},
			"multi_protocol": schema.SingleNestedAttribute{
				Description: "Multi-protocol configuration. This block will be automatically created by the FlashBlade when both NFS and SMB are enabled.",
				Optional:    true,
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"access_control_style": schema.StringAttribute{Optional: true, Computed: true},
					"safeguard_acls":       schema.BoolAttribute{Optional: true, Computed: true},
				},
			},
		},
	}
}

// Map FB API filesystem to resource model
func mapFileSystemToModel(fs *fb.FileSystem, model *fileSystemResourceModel) {
	model.ID = types.StringPointerValue(fs.Id)
	model.Name = types.StringPointerValue(fs.Name)
	model.Provisioned = types.Int64PointerValue(fs.Provisioned)
	model.HardLimitEnabled = types.BoolPointerValue(fs.HardLimitEnabled)
	model.DefaultGroupQuota = types.Int64PointerValue(fs.DefaultGroupQuota)
	model.DefaultUserQuota = types.Int64PointerValue(fs.DefaultUserQuota)
	model.SnapshotDirectoryEnabled = types.BoolPointerValue(fs.SnapshotDirectoryEnabled)
	model.Writable = types.BoolPointerValue(fs.Writable)
	model.RequestedPromotionState = types.StringPointerValue(fs.PromotionStatus)
	model.Created = types.Int64PointerValue(fs.Created)
	model.Destroyed = types.BoolPointerValue(fs.Destroyed)
	model.TimeRemaining = types.Int64PointerValue(fs.TimeRemaining)

	if fs.Nfs != nil && (fs.Nfs.V3Enabled != nil || fs.Nfs.V41Enabled != nil) {
		model.Nfs = basetypes.NewObjectValueMust(nfsAttributeTypes, map[string]attr.Value{
			"v3_enabled":   types.BoolPointerValue(fs.Nfs.V3Enabled),
			"v4_1_enabled": types.BoolPointerValue(fs.Nfs.V41Enabled),
			"rules":        types.StringPointerValue(fs.Nfs.Rules),
		})
	} else {
		model.Nfs = types.ObjectNull(nfsAttributeTypes)
	}

	if fs.Smb != nil && fs.Smb.Enabled != nil && *fs.Smb.Enabled {
		clientPolicyName := types.StringNull()
		if fs.Smb.ClientPolicy != nil {
			clientPolicyName = types.StringPointerValue(fs.Smb.ClientPolicy.Name)
		}
		sharePolicyName := types.StringNull()
		if fs.Smb.SharePolicy != nil {
			sharePolicyName = types.StringPointerValue(fs.Smb.SharePolicy.Name)
		}
		model.Smb = basetypes.NewObjectValueMust(smbAttributeTypes, map[string]attr.Value{
			"enabled":                         types.BoolPointerValue(fs.Smb.Enabled),
			"continuous_availability_enabled": types.BoolPointerValue(fs.Smb.ContinuousAvailabilityEnabled),
			"client_policy_name":              clientPolicyName,
			"share_policy_name":               sharePolicyName,
		})
	} else {
		model.Smb = types.ObjectNull(smbAttributeTypes)
	}
	
	if fs.MultiProtocol != nil {
		model.MultiProtocol = basetypes.NewObjectValueMust(multiProtocolAttributeTypes, map[string]attr.Value{
			"access_control_style": types.StringPointerValue(fs.MultiProtocol.AccessControlStyle),
			"safeguard_acls":       types.BoolPointerValue(fs.MultiProtocol.SafeguardAcls),
		})
	} else {
		model.MultiProtocol = types.ObjectNull(multiProtocolAttributeTypes)
	}

	if fs.QosPolicy != nil {
		model.QosPolicyName = types.StringPointerValue(fs.QosPolicy.Name)
	} else {
		model.QosPolicyName = types.StringNull()
	}
}

// --- CREATE ---
func (r *fileSystemResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan fileSystemResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() { return }

	fsToCreate := fb.FileSystemPost{
		Provisioned:              plan.Provisioned.ValueInt64Pointer(),
		HardLimitEnabled:         plan.HardLimitEnabled.ValueBoolPointer(),
		DefaultGroupQuota:        plan.DefaultGroupQuota.ValueInt64Pointer(),
		DefaultUserQuota:         plan.DefaultUserQuota.ValueInt64Pointer(),
		SnapshotDirectoryEnabled: plan.SnapshotDirectoryEnabled.ValueBoolPointer(),
		Writable:                 plan.Writable.ValueBoolPointer(),
		RequestedPromotionState:  plan.RequestedPromotionState.ValueStringPointer(),
	}

	if !plan.Nfs.IsNull() {
		nfsData := nfsModel{}
		resp.Diagnostics.Append(plan.Nfs.As(ctx, &nfsData, basetypes.ObjectAsOptions{})...)
		if resp.Diagnostics.HasError() { return }
		fsToCreate.Nfs = &fb.Nfs{
			V3Enabled:  nfsData.V3Enabled.ValueBoolPointer(),
			V41Enabled: nfsData.V41Enabled.ValueBoolPointer(),
			Rules:      nfsData.Rules.ValueStringPointer(),
		}
	}
	
	if !plan.Smb.IsNull() {
		smbData := smbModel{}
		resp.Diagnostics.Append(plan.Smb.As(ctx, &smbData, basetypes.ObjectAsOptions{})...)
		if resp.Diagnostics.HasError() { return }
		fsToCreate.Smb = &fb.SmbPost{
			Enabled:                       smbData.Enabled.ValueBoolPointer(),
			ContinuousAvailabilityEnabled: smbData.ContinuousAvailabilityEnabled.ValueBoolPointer(),
		}
		if !smbData.ClientPolicyName.IsNull() {
			fsToCreate.Smb.ClientPolicy = &fb.ReferenceWritable{Name: smbData.ClientPolicyName.ValueStringPointer()}
		}
		if !smbData.SharePolicyName.IsNull() {
			fsToCreate.Smb.SharePolicy = &fb.ReferenceWritable{Name: smbData.SharePolicyName.ValueStringPointer()}
		}
	}

	if !plan.QosPolicyName.IsNull() {
		fsToCreate.QosPolicy = &fb.Reference{Name: plan.QosPolicyName.ValueStringPointer()}
	}

	createdFS, err := r.client.CreateFileSystem(ctx, plan.Name.ValueString(), &fsToCreate)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating File System", "Could not create file system: "+err.Error())
		return
	}
	
	mapFileSystemToModel(createdFS, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

// --- READ ---
func (r *fileSystemResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state fileSystemResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() { return }

	fs, err := r.client.GetFileSystemByName(ctx, state.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error Reading File System", fmt.Sprintf("Could not read file system %s: %s", state.Name.ValueString(), err.Error()))
		return
	}
	if fs == nil || (fs.Destroyed != nil && *fs.Destroyed) {
		tflog.Warn(ctx, "File system not found or destroyed, removing from state.", map[string]interface{}{"name": state.Name.ValueString()})
		resp.State.RemoveResource(ctx)
		return
	}
	
	mapFileSystemToModel(fs, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// --- UPDATE ---
func (r *fileSystemResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state fileSystemResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() { return }

	fsToUpdate := fb.FileSystemPatch{}
	isPatchNeeded := false

	if !plan.Provisioned.Equal(state.Provisioned) { isPatchNeeded = true; fsToUpdate.Provisioned = plan.Provisioned.ValueInt64Pointer() }
	if !plan.HardLimitEnabled.Equal(state.HardLimitEnabled) { isPatchNeeded = true; fsToUpdate.HardLimitEnabled = plan.HardLimitEnabled.ValueBoolPointer() }
	if !plan.DefaultGroupQuota.Equal(state.DefaultGroupQuota) { isPatchNeeded = true; fsToUpdate.DefaultGroupQuota = plan.DefaultGroupQuota.ValueInt64Pointer() }
	if !plan.DefaultUserQuota.Equal(state.DefaultUserQuota) { isPatchNeeded = true; fsToUpdate.DefaultUserQuota = plan.DefaultUserQuota.ValueInt64Pointer() }
	if !plan.SnapshotDirectoryEnabled.Equal(state.SnapshotDirectoryEnabled) { isPatchNeeded = true; fsToUpdate.SnapshotDirectoryEnabled = plan.SnapshotDirectoryEnabled.ValueBoolPointer() }
	if !plan.Writable.Equal(state.Writable) { isPatchNeeded = true; fsToUpdate.Writable = plan.Writable.ValueBoolPointer() }
	if !plan.RequestedPromotionState.Equal(state.RequestedPromotionState) { isPatchNeeded = true; fsToUpdate.RequestedPromotionState = plan.RequestedPromotionState.ValueStringPointer() }

	if !plan.QosPolicyName.Equal(state.QosPolicyName) {
		isPatchNeeded = true
		if plan.QosPolicyName.IsNull() { fsToUpdate.QosPolicy = &fb.Reference{Name: types.StringValue("").ValueStringPointer()} } else { fsToUpdate.QosPolicy = &fb.Reference{Name: plan.QosPolicyName.ValueStringPointer()} }
	}

	if !plan.Nfs.Equal(state.Nfs) {
		isPatchNeeded = true
		var planNfs nfsModel
		if !plan.Nfs.IsNull() {
			resp.Diagnostics.Append(plan.Nfs.As(ctx, &planNfs, basetypes.ObjectAsOptions{})...)
			if resp.Diagnostics.HasError() { return }
			fsToUpdate.Nfs = &fb.NfsPatch{
				V3Enabled:  planNfs.V3Enabled.ValueBoolPointer(),
				V41Enabled: planNfs.V41Enabled.ValueBoolPointer(),
				Rules:      planNfs.Rules.ValueStringPointer(),
			}
		} else {
			fsToUpdate.Nfs = &fb.NfsPatch{ V3Enabled: types.BoolValue(false).ValueBoolPointer(), V41Enabled: types.BoolValue(false).ValueBoolPointer(), Rules: types.StringValue("").ValueStringPointer() }
		}
	}

	if !plan.Smb.Equal(state.Smb) {
		isPatchNeeded = true
		var planSmb smbModel
		if !plan.Smb.IsNull() {
			resp.Diagnostics.Append(plan.Smb.As(ctx, &planSmb, basetypes.ObjectAsOptions{})...)
			if resp.Diagnostics.HasError() { return }
			fsToUpdate.Smb = &fb.Smb{
				Enabled:                       planSmb.Enabled.ValueBoolPointer(),
				ContinuousAvailabilityEnabled: planSmb.ContinuousAvailabilityEnabled.ValueBoolPointer(),
			}
			if !planSmb.ClientPolicyName.IsNull() {
				fsToUpdate.Smb.ClientPolicy = &fb.ReferenceWritable{Name: planSmb.ClientPolicyName.ValueStringPointer()}
			}
			if !planSmb.SharePolicyName.IsNull() {
				fsToUpdate.Smb.SharePolicy = &fb.ReferenceWritable{Name: planSmb.SharePolicyName.ValueStringPointer()}
			}
		} else {
			fsToUpdate.Smb = &fb.Smb{ Enabled: types.BoolValue(false).ValueBoolPointer() }
		}
	}
	
	if !isPatchNeeded {
		tflog.Debug(ctx, "No changes detected for file system, skipping API call.")
		return
	}

	updatedFS, err := r.client.UpdateFileSystem(ctx, plan.Name.ValueString(), &fsToUpdate)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating File System", fmt.Sprintf("Could not update file system: %s", err.Error()))
		return
	}

	mapFileSystemToModel(updatedFS, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}


// --- DELETE ---
func (r *fileSystemResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state fileSystemResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() { return }

	fsName := state.Name.ValueString()

	fs, err := r.client.GetFileSystemByName(ctx, fsName)
	if err != nil {
		resp.Diagnostics.AddError("Error Checking File System on Delete", fmt.Sprintf("Could not read file system %s before deletion: %s", fsName, err.Error()))
		return
	}
	if fs == nil {
		tflog.Warn(ctx, "File system not found, removing from state.", map[string]interface{}{"name": fsName})
		return
	}

	if fs.Destroyed == nil || !*fs.Destroyed {
		tflog.Debug(ctx, "Step 1: Disabling protocols and marking for destruction...", map[string]interface{}{"name": fsName})
		shouldDestroy := true
		shouldDisable := false
		patch := fb.FileSystemPatch{
			Destroyed: &shouldDestroy,
			Nfs:       &fb.NfsPatch{V3Enabled: &shouldDisable, V41Enabled: &shouldDisable},
			Smb:       &fb.Smb{Enabled: &shouldDisable},
		}
		_, err = r.client.UpdateFileSystem(ctx, fsName, &patch)
		if err != nil {
			resp.Diagnostics.AddError("Error Marking File System For Deletion", fmt.Sprintf("Could not disable protocols and mark file system %s for deletion: %s", fsName, err.Error()))
			return
		}
	} else {
		tflog.Debug(ctx, "File system is already marked for destruction. Skipping soft delete step.", map[string]interface{}{"name": fsName})
	}

	tflog.Debug(ctx, "Step 2: Eradicating the file system...", map[string]interface{}{"name": fsName})
	err = r.client.EradicateFileSystem(ctx, fsName)
	if err != nil {
		resp.Diagnostics.AddError("Error Eradicating File System", fmt.Sprintf("Could not eradicate file system %s: %s", fsName, err.Error()))
		return
	}
}

// --- CONFIGURE ---
func (r *fileSystemResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil { return }
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", fmt.Sprintf("Expected *client.Client, got: %T.", req.ProviderData))
		return
	}
	r.client = c
}

// --- IMPORT ---
func (r *fileSystemResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}
