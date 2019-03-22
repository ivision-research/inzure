The examples in this directory are intended to give an idea of how inzure could fit into a pipeline. They are as follows:

- [fixit](fixit) - fixit actually uses the Azure API to fix problems it discovers directly. This is a fully automated discovery to mitigation system.
- [slackposter](slackposter) - slackposter will send messages to Slack when it finds a configuration error. It does not actually fix the problem, but makes it fairly obvious.

These examples are all fairly minimal to show conceptual usage. They all only rely on inzure query strings for configuration error identification. More in depth tests can be used in conjunction with the inzure testing framework to do more complex tasks.
