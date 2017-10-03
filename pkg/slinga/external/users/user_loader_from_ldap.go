package users

import (
	"fmt"
	"github.com/Aptomi/aptomi/pkg/slinga/db"
	"github.com/Aptomi/aptomi/pkg/slinga/lang"
	"github.com/Aptomi/aptomi/pkg/slinga/lang/yaml"
	"github.com/Aptomi/aptomi/pkg/slinga/util"
	"github.com/mattn/go-zglob"
	"gopkg.in/ldap.v2"
	"strconv"
	"strings"
	"sync"
)

// LDAPConfig contains configuration for LDAP sync service (host, port, DN, filter query and mapping of LDAP properties to Aptomi attributes)
type LDAPConfig struct {
	Host              string
	Port              int
	BaseDN            string
	Filter            string
	LabelToAtrributes map[string]string
}

// Loads LDAP configuration
func loadLDAPConfig(baseDir string) *LDAPConfig {
	files, _ := zglob.Glob(db.GetAptomiObjectFilePatternYaml(baseDir, db.TypeUsersLDAP))
	fileName, err := util.EnsureSingleFile(files)
	if err != nil {
		panic(fmt.Sprintf("LDAP config lookup error in directory '%s': %s", baseDir, err.Error()))
	}
	result := yaml.LoadObjectFromFile(fileName, &LDAPConfig{}).(*LDAPConfig)
	return result
}

// Returns the list of attributes to be retrieved from LDAP
func (config *LDAPConfig) getAttributes() []string {
	result := []string{}
	for _, attr := range config.LabelToAtrributes {
		result = append(result, attr)
	}
	return result
}

// UserLoaderFromLDAP allows aptomi to load users from LDAP
type UserLoaderFromLDAP struct {
	once sync.Once

	baseDir     string
	config      *LDAPConfig
	cachedUsers *lang.GlobalUsers
}

// NewUserLoaderFromLDAP returns new UserLoaderFromLDAP, given location with LDAP configuration file (with host/port and mapping)
func NewUserLoaderFromLDAP(baseDir string) UserLoader {
	return &UserLoaderFromLDAP{
		baseDir: baseDir,
		config:  loadLDAPConfig(baseDir),
	}
}

// LoadUsersAll loads all users
func (loader *UserLoaderFromLDAP) LoadUsersAll() lang.GlobalUsers {
	// Right now this can be called concurrently by the engine, so it needs to be thread safe
	loader.once.Do(func() {
		loader.cachedUsers = &lang.GlobalUsers{Users: make(map[string]*lang.User)}
		t := loader.ldapSearch()
		for _, u := range t {
			// add user
			loader.cachedUsers.Users[u.ID] = u
		}
	})
	return *loader.cachedUsers
}

// LoadUserByID loads a single user by ID
func (loader *UserLoaderFromLDAP) LoadUserByID(id string) *lang.User {
	return loader.LoadUsersAll().Users[id]
}

// Summary returns summary as string
func (loader *UserLoaderFromLDAP) Summary() string {
	return strconv.Itoa(len(loader.LoadUsersAll().Users)) + " (from LDAP)"
}

// Does search on LDAP and returns entries
func (loader *UserLoaderFromLDAP) ldapSearch() []*lang.User {
	l, err := ldap.Dial("tcp", fmt.Sprintf("%s:%d", loader.config.Host, loader.config.Port))
	if err != nil {
		panic(err)
	}
	defer l.Close()

	searchRequest := ldap.NewSearchRequest(
		loader.config.BaseDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		loader.config.Filter,
		loader.config.getAttributes(),
		nil,
	)

	searchResult, err := l.Search(searchRequest)
	if err != nil {
		panic(err)
	}

	result := []*lang.User{}
	for _, entry := range searchResult.Entries {
		user := &lang.User{
			ID:     entry.DN,
			Name:   entry.GetAttributeValue(loader.config.LabelToAtrributes["name"]),
			Labels: make(map[string]string),
		}
		for label, attr := range loader.config.LabelToAtrributes {
			if label != "id" && label != "name" {
				value := entry.GetAttributeValue(attr)
				if len(value) > 0 {
					user.Labels[label] = ldapValue(value)
				}
			}
		}

		// fmt.Printf("%+v\n", user)
		result = append(result, user)
	}

	return result
}

func ldapValue(value string) string {
	// normalize boolean values
	if strings.ToLower(value) == "true" {
		return "true"
	}
	if strings.ToLower(value) == "false" {
		return "false"
	}
	return value
}
