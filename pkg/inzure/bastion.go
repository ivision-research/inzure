package inzure

import (
	armnetwork "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v8"
)

//go:generate go run gen/enum.go -type-name ProvisioningState -prefix Provisioning -values Canceled,Creating,Deleting,Failed,Succeeded,Updating -azure-type ProvisioningState -azure-values ProvisioningStateCanceled,ProvisioningStateCreating,ProvisioningStateDeleting,ProvisioningStateFailed,ProvisioningStateSucceeded,ProvisioningStateUpdating -azure-import github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v8 -no-string

type BastionHostIPConfiguration struct {
	Meta       ResourceID
	SubnetMeta ResourceID
	PublicIP   PublicIP
	Name       string
}

func (ipc *BastionHostIPConfiguration) FromAzure(az *armnetwork.BastionHostIPConfiguration) {
	if az.ID == nil {
		return
	}
	ipc.Meta.FromID(*az.ID)
	gValFromPtr(&ipc.Name, az.Name)

	if props := az.Properties; props != nil {
		if props.Subnet != nil {
			gValFromPtrFromAzure(&ipc.SubnetMeta, props.Subnet.ID)
		}

		if props.PublicIPAddress != nil {
			gValFromPtrFromAzure(&ipc.PublicIP.Meta, props.PublicIPAddress.ID)
		}
	}
}

type BastionHost struct {
	Meta ResourceID

	IPConfigurations []BastionHostIPConfiguration

	Endpoint string

	FileCopyEnabled         UnknownBool
	IPConnectEnabled        UnknownBool
	KerberosEnabled         UnknownBool
	SessionRecordingEnabled UnknownBool
	ShareableLinkEnabled    UnknownBool
	TunnelingEnabled        UnknownBool

	NetworkACLs IPCollection

	PrivateOnly UnknownBool

	State ProvisioningState
}

func NewEmptyBastionHost() *BastionHost {
	var rid ResourceID
	rid.setupEmpty()
	return &BastionHost{
		Meta:                    rid,
		IPConfigurations:        make([]BastionHostIPConfiguration, 0),
		State:                   ProvisioningUnknown,
		NetworkACLs:             make([]AzureIPv4, 0),
		FileCopyEnabled:         BoolUnknown,
		IPConnectEnabled:        BoolUnknown,
		KerberosEnabled:         BoolUnknown,
		SessionRecordingEnabled: BoolUnknown,
		ShareableLinkEnabled:    BoolUnknown,
		TunnelingEnabled:        BoolUnknown,
		PrivateOnly:             BoolUnknown,
	}
}

func (bh *BastionHost) FromAzure(az *armnetwork.BastionHost) {
	if az.ID == nil {
		return
	}
	bh.Meta.FromID(*az.ID)
	props := az.Properties
	if props == nil {
		return
	}
	gValFromPtr(&bh.Endpoint, props.DNSName)
	bh.FileCopyEnabled.FromBoolPtr(props.EnableFileCopy)
	bh.IPConnectEnabled.FromBoolPtr(props.EnableIPConnect)
	bh.KerberosEnabled.FromBoolPtr(props.EnableKerberos)
	bh.PrivateOnly.FromBoolPtr(props.EnablePrivateOnlyBastion)
	bh.SessionRecordingEnabled.FromBoolPtr(props.EnableSessionRecording)
	bh.ShareableLinkEnabled.FromBoolPtr(props.EnableShareableLink)
	bh.TunnelingEnabled.FromBoolPtr(props.EnableTunneling)

	bh.State.FromAzure(props.ProvisioningState)
	if acls := props.NetworkACLs; acls != nil && len(acls.IPRules) > 0 {
		rules := acls.IPRules
		for _, ip := range rules {
			if ip != nil && ip.AddressPrefix != nil {
				bh.NetworkACLs = append(bh.NetworkACLs, NewAzureIPv4FromAzure(*ip.AddressPrefix))
			}
		}
	}

	if ipcs := props.IPConfigurations; len(ipcs) > 0 {
		for _, ipc := range ipcs {
			var bhipc BastionHostIPConfiguration
			bhipc.FromAzure(ipc)
			bh.IPConfigurations = append(bh.IPConfigurations, bhipc)
		}
	}
}
