# fixit

fixit is the most aggressive use of inzure: it uses inzure as a pipeline to actually fix discovered misconfigurations automatically. Note that you could easily break things that depend on misconfigurations. Be certain you're fixing something that you deem critical and worth breaking a product over if you choose a pipeline similar to this example.

The example provided here finds all Redis servers that support unencrypted communication and attempts to fix that via the Azure API.
