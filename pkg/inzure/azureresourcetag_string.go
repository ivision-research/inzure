// Code generated by "stringer -type=AzureResourceTag"; DO NOT EDIT.

package inzure

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[ResourceUnsetT-0]
	_ = x[ResourceUnknownT-1]
	_ = x[ResourceGroupT-2]
	_ = x[StorageAccountT-3]
	_ = x[ContainerT-4]
	_ = x[QueueT-5]
	_ = x[FileShareT-6]
	_ = x[TableT-7]
	_ = x[ProviderT-8]
	_ = x[NetworkSecurityGroupT-9]
	_ = x[VirtualNetworkT-10]
	_ = x[VirtualMachineT-11]
	_ = x[SubnetT-12]
	_ = x[NetworkInterfaceT-13]
	_ = x[IPConfigurationT-14]
	_ = x[PublicIPT-15]
	_ = x[WebAppT-16]
	_ = x[FunctionT-17]
	_ = x[DataLakeT-18]
	_ = x[DataLakeStoreT-19]
	_ = x[DataLakeAnalyticsT-20]
	_ = x[SQLServerT-21]
	_ = x[WebAppSlotT-22]
	_ = x[RedisServerT-23]
	_ = x[RecommendationT-24]
	_ = x[SQLDatabaseT-25]
	_ = x[VirtualMachineScaleSetT-26]
	_ = x[ApiT-27]
	_ = x[ApiServiceT-28]
	_ = x[ApiOperationT-29]
	_ = x[ApiBackendT-30]
	_ = x[ApiServiceProductT-31]
	_ = x[ServiceBusT-32]
	_ = x[ServiceFabricT-33]
	_ = x[ApiSchemaT-34]
	_ = x[LoadBalancerT-35]
	_ = x[FrontendIPConfigurationT-36]
	_ = x[ApplicationSecurityGroupT-37]
	_ = x[KeyVaultT-38]
	_ = x[CosmosDBT-39]
	_ = x[PostgresServerT-40]
	_ = x[PostgresDBT-41]
}

const _AzureResourceTag_name = "ResourceUnsetTResourceUnknownTResourceGroupTStorageAccountTContainerTQueueTFileShareTTableTProviderTNetworkSecurityGroupTVirtualNetworkTVirtualMachineTSubnetTNetworkInterfaceTIPConfigurationTPublicIPTWebAppTFunctionTDataLakeTDataLakeStoreTDataLakeAnalyticsTSQLServerTWebAppSlotTRedisServerTRecommendationTSQLDatabaseTVirtualMachineScaleSetTApiTApiServiceTApiOperationTApiBackendTApiServiceProductTServiceBusTServiceFabricTApiSchemaTLoadBalancerTFrontendIPConfigurationTApplicationSecurityGroupTKeyVaultTCosmosDBTPostgresServerTPostgresDBT"

var _AzureResourceTag_index = [...]uint16{0, 14, 30, 44, 59, 69, 75, 85, 91, 100, 121, 136, 151, 158, 175, 191, 200, 207, 216, 225, 239, 257, 267, 278, 290, 305, 317, 340, 344, 355, 368, 379, 397, 408, 422, 432, 445, 469, 494, 503, 512, 527, 538}

func (i AzureResourceTag) String() string {
	if i >= AzureResourceTag(len(_AzureResourceTag_index)-1) {
		return "AzureResourceTag(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _AzureResourceTag_name[_AzureResourceTag_index[i]:_AzureResourceTag_index[i+1]]
}
