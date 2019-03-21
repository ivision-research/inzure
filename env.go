package inzure

var (
	// EnvSubscriptionJSON defines an environmental variable that can hold
	// a single filename referring to an inzure JSON file
	EnvSubscriptionJSON = "INZURE_JSON_FILE"
	// EnvSubscription defines an environmental variable that can hold a single
	// inzure subscription UUID and optional alias. To specify an alias, use an
	// = after the UUID.
	EnvSubscription = "INZURE_SUBSCRIPTION"
	// EnvSubscriptionFile defines an environmental variable that can hold a
	// file containing a newline separated list of subscription UUIDs and
	// optional aliases.
	EnvSubscriptionFile = "INZURE_SUBSCRIPTION_FILE"
)
