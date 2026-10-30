package review

import "testing"

func TestPolicyCannotActivateUnapprovedSkillsOrTools(t *testing.T) {
	project := &Project{AllowedSkills: []string{"vercel-react-best-practices"}}
	for _, raw := range []string{"version: 1\nskills: [arbitrary-skill]\n", "version: 1\ntools: [terminal]\n", "version: 1\n---\ntools: [terminal]\n"} {
		if _, err := ParsePolicy(raw, project); err == nil {
			t.Fatal("unapproved capability accepted")
		}
	}
	if _, err := ParsePolicy("version: 1\nskills: [vercel-react-best-practices]\ninstructions: Follow the target branch guide\n", project); err != nil {
		t.Fatal(err)
	}
}
