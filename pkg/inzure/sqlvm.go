package inzure

import (
	azsqlvm "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/sqlvirtualmachine/armsqlvirtualmachine"
)

type SQLVirtualMachine struct {
	Meta                      ResourceID
	GroupResourceId           ResourceID
	AutoUpgrade               UnknownBool
	AutoPatch                 UnknownBool
	AutoBackup                UnknownBool
	EncryptAutoBackup         UnknownBool
	BackupSystemDBs           UnknownBool
	AssessmentEnabled         UnknownBool
	AssessmentScheduleEnabled UnknownBool
	KeyVaultEnabled           UnknownBool
	LeastPrivilegeModeEnabled UnknownBool
	IsPublic                  UnknownBool
	Port                      AzurePort
	MangementType             string
	KeyVaultServicePrincipal  string
	KeyVaultURL               string
	KeyVaultCredentialName    string
	StorageAccountURL         string
	StorageContainerName      string
}

func NewEmptySQLVirtualMachine() *SQLVirtualMachine {
	var rid ResourceID
	rid.setupEmpty()
	return &SQLVirtualMachine{
		Meta:            rid,
		GroupResourceId: rid,
		Port:            NewPortFromUint16(uint16(1433)),
	}
}

func (it *SQLVirtualMachine) FromAzure(az *azsqlvm.SQLVirtualMachine) {
	if az.ID == nil {
		return
	}
	it.Meta.FromID(*az.ID)
	props := az.Properties
	if props == nil {
		return
	}

	it.AutoUpgrade.FromBoolPtr(props.EnableAutomaticUpgrade)
	if aps := props.AutoPatchingSettings; aps != nil {
		it.AutoPatch.FromBoolPtr(aps.Enable)
	}

	if abs := props.AutoBackupSettings; abs != nil {
		it.AutoBackup.FromBoolPtr(abs.Enable)
		it.EncryptAutoBackup.FromBoolPtr(abs.EnableEncryption)
		it.BackupSystemDBs.FromBoolPtr(abs.BackupSystemDbs)
		gValFromPtr(&it.StorageAccountURL, abs.StorageAccountURL)
		gValFromPtr(&it.StorageContainerName, abs.StorageContainerName)
	}

	if as := props.AssessmentSettings; as != nil {
		it.AssessmentEnabled.FromBoolPtr(as.Enable)
		if sched := as.Schedule; sched != nil {
			it.AssessmentScheduleEnabled.FromBoolPtr(sched.Enable)
		}
	}

	if kv := props.KeyVaultCredentialSettings; kv != nil {
		it.KeyVaultEnabled.FromBoolPtr(kv.Enable)
		gValFromPtr(&it.KeyVaultURL, kv.AzureKeyVaultURL)
		gValFromPtr(&it.KeyVaultCredentialName, kv.CredentialName)
		gValFromPtr(&it.KeyVaultServicePrincipal, kv.ServicePrincipalName)
	}

	if props.LeastPrivilegeMode != nil {
		it.LeastPrivilegeModeEnabled.FromBool(*props.LeastPrivilegeMode == azsqlvm.LeastPrivilegeModeEnabled)
	}

	if m := props.SQLManagement; m != nil {
		it.MangementType = string(*m)
	}

	if gid := props.SQLVirtualMachineGroupResourceID; gid != nil {
		it.GroupResourceId.FromID(*gid)
	}

	if scm := props.ServerConfigurationsManagementSettings; scm != nil {
		if cs := scm.SQLConnectivityUpdateSettings; cs != nil {
			if cs.ConnectivityType != nil {
				it.IsPublic.FromBool(*cs.ConnectivityType == azsqlvm.ConnectivityTypePUBLIC)
			}

			if cs.Port != nil {
				it.Port = NewPortFromUint16(uint16(*cs.Port))
			}
		}
	}
}
