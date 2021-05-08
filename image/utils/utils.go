package utils

const (
	Latest = "latest"
)

// image name would not contain scheme like http https
/*func imageHostName(imageName string) string {
	//TODO strengthen the image host verification
	ind := strings.IndexRune(imageName, '/')
	if ind >= 0 && strings.ContainsAny(imageName[0:ind], ".:") {
		return imageName[0:ind]
	}
	return ""
}*/

// input: urlImageName could be like "***.com/k8s:v1.1" or "k8s:v1.1"
// output: like "k8s:v1.1"
/*func repoAndTag(imageName string) (string, string) {
	newImageName := strings.TrimPrefix(
		strings.TrimPrefix(imageName, imageHostName(imageName)),
		"/")
	splits := strings.Split(newImageName, ":")
	repo, tag := newImageName, Latest
	if len(splits) == 2 {
		repo = splits[0]
		tag = splits[1]
	}
	return repo, tag
}*/

// input: urlImageName could be like "***.com/k8s:v1.1" or "k8s:v1.1"
// output: like "***.com/k8s"
/*func rawRepo(imageName string) string {
	repo := imageName
	splits := strings.Split(imageName, ":")
	if len(splits) == 2 {
		repo = splits[0]
	}
	return repo
}*/
