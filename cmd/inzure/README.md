# inzure

Automated data consolidation for Azure. The goal of this tool is very simple: gather all data you'll want for an assessment of the security of an Azure cloud subscription. This tool does that and only that. The data is given to you as a large JSON structure. Other tools are designed to analyse that data or transform it as you wish. You can also use the data structures defined in the package at the base of this repository to create your own tool.

## Output Data

Note that the data output by this tool can be sensitive, potentially including passwords and keys, all firewall data for your entire Azure subscription, and every available public host name and IP as well as the information required to exploit those things. We **strongly recommended** encrypting this data and never storing it somewhere public or in source control. If you're uncertain about encrypting/decrypting the data note that this tool offers option to encrypt your data for you. If the environmental variable `INZURE_ENCRYPT_PASSWORD` is set, it should be used by all `inzure` native applications for encrypting/decrypting result data. Remember to use a sufficiently random password! If you're deciding on whether the built in encryption is good enough for you, see the [base documentation](../../README.md).

## Setup

This help documentation for authorizing access is requires the Azure CLI. If you do not have the Azure CLI, you can read about how to install it [here](https://docs.microsoft.com/en-us/cli/azure/install-azure-cli?view=azure-cli-latest). If you follow the directions given here exactly you should not run in to issues.

If you intend to use the `add_subscription.py` script, you first need to clear your Azure subscription cache with:

```
az account clear
```

This is important because that script will use all available subscriptions and we want to make sure we're only using the ones we specifically want.

Then you'll login using:

```.sh
az login
```

Next, you will create a [custom role](https://docs.microsoft.com/en-us/azure/active-directory/role-based-access-control-custom-roles) to use with [Role Based Access Control](https://docs.microsoft.com/en-us/azure/active-directory/role-based-access-control-configure). The [role.json](role.json) file defines this new role with the permissions this tool needs. You need to add your subscription ID to this role: you can either do this manually or we have provided an executable [python script](add_subscription.py) that can do it for you (note this will add all subscriptions that are in Azure's local cache, this is why we cleared it earlier). After updating the file or running the script, run:

```.sh
az role definition create --role-definition @role.json
```

This creates a role called "Inzure Tool". A role is just a collection of permissions with a name; now we need to actually apply those permissions. To do that you'll need to [create a new Service
Principle](https://docs.microsoft.com/en-us/cli/azure/create-an-azure-service-principal-azure-cli?view=azure-cli-latest) that uses this role. The command is:

```.sh
az ad sp create-for-rbac \
    --name "inzure" \
    --role "Inzure Tool" \
    --sdk-auth > credentials.json
```

That's it. Your Azure account should now be set up to run all tests included with this tool. When running be sure to set the proper environmental variables from this file:

- `AZURE_TENANT_ID`
- `AZURE_CLIENT_ID`
- `AZURE_CLIENT_SECRET`

```.sh
export AZURE_TENANT_ID={fromfile}
export AZURE_CLIENT_ID={fromfile}
export AZURE_CLIENT_SECRET={fromfile}
```

### Enabling Classic Resources

If you have classic resources on your Azure account the above setup isn't enough. You will also need to upload a management certificate. First you'll need to [follow these steps](https://docs.microsoft.com/en-us/azure/cloud-services/cloud-services-certs-create#create-a-new-self-signed-certificate) to create one if you haven't already and then [follow these steps](https://docs.microsoft.com/en-us/azure/azure-api-management-certs) to upload it to the appropriate service.

For creating the key on macOS or Linux run the following (set the -subj fields appropriately):

```
openssl req -x509 -nodes -new -newkey rsa:2048 -days 365 \
        -out cert.pem -keyout cert.pem \
        -subj "/C=US/ST=CO/L=Loc/O=Org/CN=local.localhost"
```

and then create a `.cer` file:

```
openssl x509 -inform pem -in cert.pem -outform der -out cert.cer
```

That `.cer` file is what you upload and the `.pem` is what you use to authenticate.

Then you'll run the tool with `inzure -cert=/path/to/cert.pem`. Tagging as supplied by `-tags` is still respected for classic items.
