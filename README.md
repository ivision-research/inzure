# inzure

This is the library portion used by the tools specified in the `cmd`  directories. For the main tool, see the [README here](cmd/inzure/README.md). That README covers setup and some basic documentation.

The APIs exposed here are intended to allow READ ONLY views into Azure subscriptions.

## Subscriptions

The main item for any inzure based program is the `Subscription` type. You can use this to gather data or to load already gathered data. This type also has methods for querying the data directly with [query strings](QUERY_STRINGS.md).

## Data

The primary purpose of this library is dealing with an Azure Subscription. The interface to an Azure Subscription is the `Subscription` struct and associated JSON files. The data in these JSON files is not intended to be parsed by people. Often, Go pseudo enums are used so you'll see a ton of integer values that are only meaningful when loaded back into a `Subscription`. In general, you should write Go programs that make use of this library and ingest the JSON to figure out what you're looking for.

Most of the data is "security focused" in that it would be useful to anyone performing a security audit of an Azure subscription. Much  of it is directly usable, but some is only indirectly useful. You need to know what you're looking for and why you're looking for it to make good use of this data.

## Azure Environments

Azure has different endpoints and flows for different environments. If you're not working in the default environment you'll need to export the AZURE_ENVIRONMENT variable as one of:

- AZURECHINACLOUD
- AZUREGERMANCLOUD
- AZUREPUBLICCLOUD
- AZUREUSGOVERNMENTCLOUD

## Encrypting

Every tool built using this package should allow for encrypted JSONs to be  marshaled into Subscriptions via the built in functions:  `EncryptSubscriptionAsJSON` and `SubscriptionFromEncryptedJSON`. The general  method involved is to simply inform the user that an environmental variable -  located at `inzure.KeyEnvironmentalVariableName` in this package with value  `INZURE_ENCRYPT_PASSWORD` - can be used as a password to encrypt/decrypt the  data. An encrypted JSON should be identifiable by putting  `EncryptedFileExtension` at the end (this is `.enc`).

The encryption provided by this tool works as follows:

1. PBKDF2 is used to turn your password into a 32 byte key. We use an 8 byte  salt generated with the `crypto/rand` function. This salt is the first 8  bytes of the output file and is therefore not a secret. 10,000 iterations of SHA-256 are used. You can set this higher with `INZURE_PBKDF2_ROUNDS` if you want (setting it lower will not work).
2. The output JSON is encrypted with 256 bit AES in CBC mode with the IV as the first block of cipher text.
3. The entire cipher text (including the IV) and the key are used to create an  HMAC using SHA-256.

So the output file looks like:

```
[ 8 byte PBKDF2 salt ] [ 32 byte HMAC ] [ 16 byte IV ] [ ... encrypted JSON ... ]
```
