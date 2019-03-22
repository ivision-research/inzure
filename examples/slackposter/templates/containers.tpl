The following Containers may allow public access but shouldn't:

{{ range . }}
- /StorageAccounts/{{.StorageAccount.ResourceGroupName }}/{{ .StorageAccount.Name }}/{{ .Name }} (Access level: {{ .Access }})
{{- end }}
