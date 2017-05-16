package slinga

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestLoadUsers(t *testing.T) {
	users := LoadUsersFromDir("testdata/fake")
	assert.Equal(t, 2, len(users.Users), "Two users should be loaded");
	assert.Equal(t, "Alice", users.Users["1"].Name, "Alice user should be loaded");
	assert.Equal(t, "Bob", users.Users["2"].Name, "Bob user should be loaded");

	userAlice := LoadUserByIDFromDir("testdata/fake", "1")
	assert.Equal(t, "Alice", userAlice.Name, "Should load Alice user by ID");
	assert.Equal(t, "yes", userAlice.Labels["dev"], "Alice should have dev='yes' label");
	assert.Equal(t, "no", userAlice.Labels["prod"], "Alice should have prod='no' label");
}
