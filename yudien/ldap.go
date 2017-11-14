package yudien

import (
	"fmt"
	"strconv"

	"gopkg.in/ldap.v2"
)

type LdapUser struct {
	IsAuthenticated bool
	Error           string

	Username string
	Groups   []string

	FirstName string
	FullName  string
	Email     string

	HomeDir string
	Uid     int
}

func LdapLogin(username string, password string) LdapUser {
	// Set up return value, we can return any time
	ldap_user := LdapUser{}
	ldap_user.Username = username

	ldapHost := fmt.Sprintf("%s:%d", Ldap.Host, Ldap.Port)
	fmt.Printf("LDAP: %s\n", ldapHost)

	l, err := ldap.Dial("tcp", ldapHost)
	if err != nil {
		ldap_user.IsAuthenticated = false
		ldap_user.Error = err.Error()
		return ldap_user
	}
	defer l.Close()

	fmt.Printf("Dial complete\n")

	sbr := ldap.SimpleBindRequest{
		Username: Ldap.LoginDN,
		Password: Ldap.Password,
	}
	_, err = l.SimpleBind(&sbr)
	if err != nil {
		ldap_user.IsAuthenticated = false
		ldap_user.Error = err.Error()
		return ldap_user
	}

	fmt.Printf("Bind complete\n")

	// Get User account

	filter := fmt.Sprintf("(uid=%s)", username)
	fmt.Printf("Filter: %s\n", filter)

	//TODO(g): Get these from JSON or something?  Not sure...  Probably JSON.  This is all ghetto, but it keeps things mostly anonymous and flexible
	attributes := []string{"cn", "gidNumber", "givenName", "homeDirectory", "loginShell", "mail", "sn", "uid", "uidNumber", "userPassword"}

	sr := ldap.NewSearchRequest(Ldap.UserSearch, ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false, filter, attributes, nil)
	user_result, err := l.Search(sr)
	if err != nil {
		ldap_user.IsAuthenticated = false
		ldap_user.Error = err.Error()
		return ldap_user
	}

	fmt.Printf("User Search complete: %d\n", len(user_result.Entries))

	for count, first := range user_result.Entries {

		//username = first.GetAttributeValue("sn")

		fmt.Printf("User %d: %s\n", count, first.DN)

		// Populate the result
		ldap_user.FirstName = first.GetAttributeValue("givenName")
		ldap_user.Email = first.GetAttributeValue("mail")
		ldap_user.FullName = first.GetAttributeValue("cn")
		ldap_user.Uid, _ = strconv.Atoi(first.GetAttributeValue("uidNumber"))

		for _, attr := range attributes {
			fmt.Printf("    %s == %v\n", attr, first.GetAttributeValue(attr))
		}

	}

	// Get group info for User

	filter = "(cn=*)"
	fmt.Printf("Group Filter: %s\n", filter)

	//TODO(g): Get these from JSON or something?  Not sure...  Probably JSON.  This is all ghetto, but it keeps things mostly anonymous and flexible
	attributes = []string{"cn", "gidNumber", "memberUid"}

	sr = ldap.NewSearchRequest(Ldap.GroupSearch, ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false, filter, attributes, nil)
	group_result, err := l.Search(sr)
	if err != nil {
		ldap_user.IsAuthenticated = false
		ldap_user.Error = err.Error()
		return ldap_user
	}

	fmt.Printf("Group Search complete: %d\n", len(group_result.Entries))

	user_groups := make([]string, 0)

	for count, first := range group_result.Entries {

		fmt.Printf("Group %d: %s\n", count, first.DN)

		group := first.GetAttributeValue("cn")
		group_users := first.GetAttributeValues("memberUid")

		for _, group_user := range group_users {
			if group_user == username {
				user_groups = append(user_groups, group)
			}
		}
	}

	fmt.Printf("User: %s  Groups: %v\n", username, user_groups)

	// Testing password
	err = l.Bind(fmt.Sprintf("uid=%s,%s", username, Ldap.UserSearch), password)
	if err != nil {
		ldap_user.IsAuthenticated = false
		ldap_user.Error = err.Error()
		return ldap_user
	}

	fmt.Printf("Password is correct\n")

	//TODO(g): make a struct and pack this data into it:  LdapUser{}
	ldap_user.IsAuthenticated = true
	ldap_user.Groups = user_groups

	return ldap_user
}
