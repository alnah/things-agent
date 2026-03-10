package app

import thingslib "github.com/alnah/things-agent/internal/things"

func scriptListTags(bundleID, query string) string {
	return thingslib.ScriptListTags(bundleID, query)
}

func scriptAddTag(bundleID, name, parent string) string {
	return thingslib.ScriptAddTag(bundleID, name, parent)
}

func scriptEditTag(bundleID, name, newName, parent string, parentChanged bool) string {
	return thingslib.ScriptEditTag(bundleID, name, newName, parent, parentChanged)
}

func scriptDeleteTag(bundleID, name string) string {
	return thingslib.ScriptDeleteTag(bundleID, name)
}
