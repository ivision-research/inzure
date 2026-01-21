module github.com/ivision-research/inzure/pkg/inzure

go 1.24.0

toolchain go1.24.10

require (
	github.com/Azure/azure-sdk-for-go v27.0.0+incompatible
	github.com/Azure/azure-sdk-for-go/sdk/azcore v1.20.0
	github.com/Azure/azure-sdk-for-go/sdk/azidentity v1.13.1
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/apimanagement/armapimanagement v1.0.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appservice/armappservice v1.0.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute v1.0.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/cosmos/armcosmos/v2 v2.0.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/dashboard/armdashboard v1.2.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/datalake-analytics/armdatalakeanalytics v0.6.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/datalake-store/armdatalakestore v1.0.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/keyvault/armkeyvault v1.0.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/postgresql/armpostgresql v1.0.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/redis/armredis v1.0.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources v1.2.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/sql/armsql v1.0.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/sqlvirtualmachine/armsqlvirtualmachine v0.10.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage v1.1.0
	github.com/Azure/go-autorest v11.7.0+incompatible
	golang.org/x/crypto v0.45.0
)

require github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork v1.1.0 // indirect

require (
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v8 v8.0.0
	github.com/golang-jwt/jwt/v5 v5.3.0 // indirect
)

require (
	contrib.go.opencensus.io/exporter/ocagent v0.4.10 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/internal v1.11.2 // indirect
	github.com/AzureAD/microsoft-authentication-library-for-go v1.6.0 // indirect
	github.com/census-instrumentation/opencensus-proto v0.2.0 // indirect
	github.com/dgrijalva/jwt-go v3.2.0+incompatible // indirect
	github.com/golang/protobuf v1.3.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway v1.8.5 // indirect
	github.com/hashicorp/golang-lru v0.5.0 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/pkg/browser v0.0.0-20240102092130-5ac0b6a4141c // indirect
	go.opencensus.io v0.20.0 // indirect
	golang.org/x/net v0.47.0
	golang.org/x/sync v0.18.0 // indirect
	golang.org/x/sys v0.38.0 // indirect
	golang.org/x/text v0.31.0 // indirect
	google.golang.org/api v0.2.0 // indirect
	google.golang.org/genproto v0.0.0-20190307195333-5fe7a883aa19 // indirect
	google.golang.org/grpc v1.19.1 // indirect
)
