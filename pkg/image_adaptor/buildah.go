package image_adaptor

const (
	buildahEtcRegistriesConf = `
[registries.search]
registries = ['docker.io']

# Registries that do not use TLS when pulling images or uses self-signed
# certificates.
[registries.insecure]
registries = []

[registries.block]
registries = []
`

	builadhEtcPolicy = `
{
    "default": [
	{
	    "type": "insecureAcceptAnything"
	}
    ],
    "transports":
	{
	    "docker-daemon":
		{
		    "": [{"type":"insecureAcceptAnything"}]
		}
	}
}`
)

func initBuildah() error {
	policyAbsPath := "/etc/containers/policy.json"
	err := writeFileIfNotExist(policyAbsPath, []byte(builadhEtcPolicy))
	if err != nil {
		return err
	}

	registriesAbsPath := "/etc/containers/registries.conf"
	return writeFileIfNotExist(registriesAbsPath, []byte(buildahEtcRegistriesConf))
}
