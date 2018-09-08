/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package plugins

import (
	"errors"
	"fmt"
	"io/ioutil"
	"path"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/sets"

	"k8s.io/test-infra/prow/commentpruner"
	"k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/git"
	"k8s.io/test-infra/prow/github"
	"k8s.io/test-infra/prow/kube"
	"k8s.io/test-infra/prow/pluginhelp"
	"k8s.io/test-infra/prow/repoowners"
	"k8s.io/test-infra/prow/slack"
)

const (
	defaultBlunderbussReviewerCount = 2
)

var (
	pluginHelp                 = map[string]HelpProvider{}
	genericCommentHandlers     = map[string]GenericCommentHandler{}
	issueHandlers              = map[string]IssueHandler{}
	issueCommentHandlers       = map[string]IssueCommentHandler{}
	pullRequestHandlers        = map[string]PullRequestHandler{}
	pushEventHandlers          = map[string]PushEventHandler{}
	reviewEventHandlers        = map[string]ReviewEventHandler{}
	reviewCommentEventHandlers = map[string]ReviewCommentEventHandler{}
	statusEventHandlers        = map[string]StatusEventHandler{}
)

type HelpProvider func(config *Configuration, enabledRepos []string) (*pluginhelp.PluginHelp, error)

func HelpProviders() map[string]HelpProvider {
	return pluginHelp
}

type IssueHandler func(PluginClient, github.IssueEvent) error

func RegisterIssueHandler(name string, fn IssueHandler, help HelpProvider) {
	pluginHelp[name] = help
	issueHandlers[name] = fn
}

type IssueCommentHandler func(PluginClient, github.IssueCommentEvent) error

func RegisterIssueCommentHandler(name string, fn IssueCommentHandler, help HelpProvider) {
	pluginHelp[name] = help
	issueCommentHandlers[name] = fn
}

type PullRequestHandler func(PluginClient, github.PullRequestEvent) error

func RegisterPullRequestHandler(name string, fn PullRequestHandler, help HelpProvider) {
	pluginHelp[name] = help
	pullRequestHandlers[name] = fn
}

type StatusEventHandler func(PluginClient, github.StatusEvent) error

func RegisterStatusEventHandler(name string, fn StatusEventHandler, help HelpProvider) {
	pluginHelp[name] = help
	statusEventHandlers[name] = fn
}

type PushEventHandler func(PluginClient, github.PushEvent) error

func RegisterPushEventHandler(name string, fn PushEventHandler, help HelpProvider) {
	pluginHelp[name] = help
	pushEventHandlers[name] = fn
}

type ReviewEventHandler func(PluginClient, github.ReviewEvent) error

func RegisterReviewEventHandler(name string, fn ReviewEventHandler, help HelpProvider) {
	pluginHelp[name] = help
	reviewEventHandlers[name] = fn
}

type ReviewCommentEventHandler func(PluginClient, github.ReviewCommentEvent) error

func RegisterReviewCommentEventHandler(name string, fn ReviewCommentEventHandler, help HelpProvider) {
	pluginHelp[name] = help
	reviewCommentEventHandlers[name] = fn
}

type GenericCommentHandler func(PluginClient, github.GenericCommentEvent) error

func RegisterGenericCommentHandler(name string, fn GenericCommentHandler, help HelpProvider) {
	pluginHelp[name] = help
	genericCommentHandlers[name] = fn
}

// PluginClient may be used concurrently, so each entry must be thread-safe.
type PluginClient struct {
	GitHubClient *github.Client
	KubeClient   *kube.Client
	GitClient    *git.Client
	SlackClient  *slack.Client
	OwnersClient repoowners.Interface

	CommentPruner *commentpruner.EventClient

	// Config provides information about the jobs
	// that we know how to run for repos.
	Config *config.Config
	// PluginConfig provides plugin-specific options
	PluginConfig *Configuration

	Logger *logrus.Entry
}

type PluginAgent struct {
	PluginClient

	mut           sync.Mutex
	configuration *Configuration
}

// Configuration is the top-level serialization
// target for plugin Configuration
type Configuration struct {
	// Plugins is a map of repositories (eg "k/k") to lists of
	// plugin names.
	// TODO: Link to the list of supported plugins.
	// https://github.com/kubernetes/test-infra/issues/3476
	Plugins map[string][]string `json:"plugins,omitempty"`

	// ExternalPlugins is a map of repositories (eg "k/k") to lists of
	// external plugins.
	ExternalPlugins map[string][]ExternalPlugin `json:"external_plugins,omitempty"`

	// Owners contains configuration related to handling OWNERS files.
	Owners Owners `json:"owners,omitempty"`

	// Built-in plugins specific configuration.
	Approve              []Approve              `json:"approve,omitempty"`
	Blockades            []Blockade             `json:"blockades,omitempty"`
	Blunderbuss          Blunderbuss            `json:"blunderbuss,omitempty"`
	Cat                  Cat                    `json:"cat,omitempty"`
	CherryPickUnapproved CherryPickUnapproved   `json:"cherry_pick_unapproved,omitempty"`
	ConfigUpdater        ConfigUpdater          `json:"config_updater,omitempty"`
	Heart                Heart                  `json:"heart,omitempty"`
	Label                *Label                 `json:"label,omitempty"`
	Lgtm                 []Lgtm                 `json:"lgtm,omitempty"`
	RepoMilestone        map[string]Milestone   `json:"repo_milestone,omitempty"`
	RequireMatchingLabel []RequireMatchingLabel `json:"require_matching_label,omitempty"`
	RequireSIG           RequireSIG             `json:"requiresig,omitempty"`
	Slack                Slack                  `json:"slack,omitempty"`
	SigMention           SigMention             `json:"sigmention,omitempty"`
	Size                 *Size                  `json:"size,omitempty"`
	Triggers             []Trigger              `json:"triggers,omitempty"`
	Welcome              Welcome                `json:"welcome,omitempty"`
}

// ExternalPlugin holds configuration for registering an external
// plugin in prow.
type ExternalPlugin struct {
	// Name of the plugin.
	Name string `json:"name"`
	// Endpoint is the location of the external plugin. Defaults to
	// the name of the plugin, ie. "http://{{name}}".
	Endpoint string `json:"endpoint,omitempty"`
	// Events are the events that need to be demuxed by the hook
	// server to the external plugin. If no events are specified,
	// everything is sent.
	Events []string `json:"events,omitempty"`
}

type Blunderbuss struct {
	// ReviewerCount is the minimum number of reviewers to request
	// reviews from. Defaults to requesting reviews from 2 reviewers
	// if FileWeightCount is not set.
	ReviewerCount *int `json:"request_count,omitempty"`
	// MaxReviewerCount is the maximum number of reviewers to request
	// reviews from. Defaults to 0 meaning no limit.
	MaxReviewerCount int `json:"max_request_count,omitempty"`
	// FileWeightCount is the maximum number of reviewers to request
	// reviews from. Selects reviewers based on file weighting.
	// This and request_count are mutually exclusive options.
	FileWeightCount *int `json:"file_weight_count,omitempty"`
	// ExcludeApprovers controls whether approvers are considered to be
	// reviewers. By default, approvers are considered as reviewers if
	// insufficient reviewers are available. If ExcludeApprovers is true,
	// approvers will never be considered as reviewers.
	ExcludeApprovers bool `json:"exclude_approvers,omitempty"`
}

// Owners contains configuration related to handling OWNERS files.
type Owners struct {
	// MDYAMLRepos is a list of org and org/repo strings specifying the repos that support YAML
	// OWNERS config headers at the top of markdown (*.md) files. These headers function just like
	// the config in an OWNERS file, but only apply to the file itself instead of the entire
	// directory and all sub-directories.
	// The yaml header must be at the start of the file and be bracketed with "---" like so:
	/*
		---
		approvers:
		- mikedanese
		- thockin

		---
	*/
	MDYAMLRepos []string `json:"mdyamlrepos,omitempty"`

	// SkipCollaborators disables collaborator cross-checks and forces both
	// the approve and lgtm plugins to use solely OWNERS files for access
	// control in the provided repos.
	SkipCollaborators []string `json:"skip_collaborators,omitempty"`

	// LabelsBlackList holds a list of labels that should not be present in any
	// OWNERS file, preventing their automatic addition by the owners-label plugin.
	// This check is performed by the verify-owners plugin.
	LabelsBlackList []string `json:"labels_blacklist,omitempty"`
}

func (pa *PluginAgent) MDYAMLEnabled(org, repo string) bool {
	full := fmt.Sprintf("%s/%s", org, repo)
	for _, elem := range pa.Config().Owners.MDYAMLRepos {
		if elem == org || elem == full {
			return true
		}
	}
	return false
}

func (pa *PluginAgent) SkipCollaborators(org, repo string) bool {
	full := fmt.Sprintf("%s/%s", org, repo)
	for _, elem := range pa.Config().Owners.SkipCollaborators {
		if elem == org || elem == full {
			return true
		}
	}
	return false
}

// RequireSIG specifies configuration for the require-sig plugin.
type RequireSIG struct {
	// GroupListURL is the URL where a list of the available SIGs can be found.
	GroupListURL string `json:"group_list_url,omitempty"`
}

// SigMention specifies configuration for the sigmention plugin.
type SigMention struct {
	// Regexp parses comments and should return matches to team mentions.
	// These mentions enable labeling issues or PRs with sig/team labels.
	// Furthermore, teams with the following suffixes will be mapped to
	// kind/* labels:
	//
	// * @org/team-bugs             --maps to--> kind/bug
	// * @org/team-feature-requests --maps to--> kind/feature
	// * @org/team-api-reviews      --maps to--> kind/api-change
	// * @org/team-proposals        --maps to--> kind/design
	//
	// Note that you need to make sure your regexp covers the above
	// mentions if you want to use the extra labeling. Defaults to:
	// (?m)@kubernetes/sig-([\w-]*)-(misc|test-failures|bugs|feature-requests|proposals|pr-reviews|api-reviews)
	//
	// Compiles into Re during config load.
	Regexp string         `json:"regexp,omitempty"`
	Re     *regexp.Regexp `json:"-"`
}

// Size specifies configuration for the size plugin, defining lower bounds (in # lines changed) for each size label.
// XS is assumed to be zero.
type Size struct {
	S   int `json:"s"`
	M   int `json:"m"`
	L   int `json:"l"`
	Xl  int `json:"xl"`
	Xxl int `json:"xxl"`
}

/*
  Blockade specifies a configuration for a single blockade.blockade. The configuration for the
  blockade plugin is defined as a list of these structures. Here is an example of a complete
  yaml config for the blockade plugin that is composed of 2 Blockade structs:

	blockades:
	- repos:
	  - kubernetes-incubator
	  - kubernetes/kubernetes
	  - kubernetes/test-infra
	  blockregexps:
	  - 'docs/.*'
	  - 'other-docs/.*'
	  exceptionregexps:
	  - '.*OWNERS'
	  explanation: "Files in the 'docs' directory should not be modified except for OWNERS files"
	- repos:
	  - kubernetes/test-infra
	  blockregexps:
	  - 'mungegithub/.*'
	  exceptionregexps:
	  - 'mungegithub/DeprecationWarning.md'
	  explanation: "Don't work on mungegithub! Work on Prow!"
*/
type Blockade struct {
	// Repos are either of the form org/repos or just org.
	Repos []string `json:"repos,omitempty"`
	// BlockRegexps are regular expressions matching the file paths to block.
	BlockRegexps []string `json:"blockregexps,omitempty"`
	// ExceptionRegexps are regular expressions matching the file paths that are exceptions to the BlockRegexps.
	ExceptionRegexps []string `json:"exceptionregexps,omitempty"`
	// Explanation is a string that will be included in the comment left when blocking a PR. This should
	// be an explanation of why the paths specified are blockaded.
	Explanation string `json:"explanation,omitempty"`
}

type Approve struct {
	// Repos is either of the form org/repos or just org.
	Repos []string `json:"repos,omitempty"`
	// IssueRequired indicates if an associated issue is required for approval in
	// the specified repos.
	IssueRequired bool `json:"issue_required,omitempty"`
	// ImplicitSelfApprove indicates if authors implicitly approve their own PRs
	// in the specified repos.
	ImplicitSelfApprove bool `json:"implicit_self_approve,omitempty"`
	// LgtmActsAsApprove indicates that the lgtm command should be used to
	// indicate approval
	LgtmActsAsApprove bool `json:"lgtm_acts_as_approve,omitempty"`
	// ReviewActsAsApprove indicates that GitHub review state should be used to
	// indicate approval.
	ReviewActsAsApprove bool `json:"review_acts_as_approve,omitempty"`
}

type Lgtm struct {
	// Repos is either of the form org/repos or just org.
	Repos []string `json:"repos,omitempty"`
	// ReviewActsAsLgtm indicates that a Github review of "approve" or "request changes"
	// acts as adding or removing the lgtm label
	ReviewActsAsLgtm bool `json:"review_acts_as_lgtm,omitempty"`
}

type Cat struct {
	// Path to file containing an api key for thecatapi.com
	KeyPath string `json:"key_path,omitempty"`
}

type Label struct {
	// AdditionalLabels is a set of additional labels enabled for use
	// on top of the existing "kind/*", "priority/*", and "area/*" labels.
	AdditionalLabels []string `json:"additional_labels"`
}

type Trigger struct {
	// Repos is either of the form org/repos or just org.
	Repos []string `json:"repos,omitempty"`
	// TrustedOrg is the org whose members' PRs will be automatically built
	// for PRs to the above repos. The default is the PR's org.
	TrustedOrg string `json:"trusted_org,omitempty"`
	// JoinOrgURL is a link that redirects users to a location where they
	// should be able to read more about joining the organization in order
	// to become trusted members. Defaults to the Github link of TrustedOrg.
	JoinOrgURL string `json:"join_org_url,omitempty"`
	// OnlyOrgMembers requires PRs and/or /ok-to-test comments to come from org members.
	// By default, trigger also include repo collaborators.
	OnlyOrgMembers bool `json:"only_org_members,omitempty"`
}

type Heart struct {
	// Adorees is a list of GitHub logins for members
	// for whom we will add emojis to comments
	Adorees []string `json:"adorees,omitempty"`
}

// Milestone contains the configuration options for the milestone and
// milestonestatus plugins.
type Milestone struct {
	// ID of the github team for the milestone maintainers (used for setting status labels)
	// You can curl the following endpoint in order to determine the github ID of your team
	// responsible for maintaining the milestones:
	// curl -H "Authorization: token <token>" https://api.github.com/orgs/<org-name>/teams
	MaintainersID   int    `json:"maintainers_id,omitempty"`
	MaintainersTeam string `json:"maintainers_team,omitempty"`
}

type Slack struct {
	MentionChannels []string       `json:"mentionchannels,omitempty"`
	MergeWarnings   []MergeWarning `json:"mergewarnings,omitempty"`
}

// ConfigMapSpec contains configuration options for the configMap being updated by the ConfigUpdater plugin
type ConfigMapSpec struct {
	// Name of ConfigMap
	Name string `json:"name"`
	// Key is the key in the ConfigMap to update with the file contents.
	// If no explicit key is given, the basename of the file will be used.
	Key string `json:"key,omitempty"`
	// Namespace in which the configMap needs to be deployed. If no namespace is specified
	// it will be deployed to the ProwJobNamespace.
	Namespace string `json:"namespace,omitempty"`
}

type ConfigUpdater struct {
	// A map of filename => ConfigMapSpec.
	// Whenever a commit changes filename, prow will update the corresponding configmap.
	// map[string]ConfigMapSpec{ "/my/path.yaml": {Name: "foo", Namespace: "otherNamespace" }}
	// will result in replacing the foo configmap whenever path.yaml changes
	Maps map[string]ConfigMapSpec `json:"maps,omitempty"`
	// The location of the prow configuration file inside the repository
	// where the config-updater plugin is enabled. This needs to be relative
	// to the root of the repository, eg. "prow/config.yaml" will match
	// github.com/kubernetes/test-infra/prow/config.yaml assuming the config-updater
	// plugin is enabled for kubernetes/test-infra. Defaults to "prow/config.yaml".
	ConfigFile string `json:"config_file,omitempty"`
	// The location of the prow plugin configuration file inside the repository
	// where the config-updater plugin is enabled. This needs to be relative
	// to the root of the repository, eg. "prow/plugins.yaml" will match
	// github.com/kubernetes/test-infra/prow/plugins.yaml assuming the config-updater
	// plugin is enabled for kubernetes/test-infra. Defaults to "prow/plugins.yaml".
	PluginFile string `json:"plugin_file,omitempty"`
}

// MergeWarning is a config for the slackevents plugin's manual merge warings.
// If a PR is pushed to any of the repos listed in the config
// then send messages to the all the  slack channels listed if pusher is NOT in the whitelist.
type MergeWarning struct {
	// Repos is either of the form org/repos or just org.
	Repos []string `json:"repos,omitempty"`
	// List of channels on which a event is published.
	Channels []string `json:"channels,omitempty"`
	// A slack event is published if the user is not part of the WhiteList.
	WhiteList []string `json:"whitelist,omitempty"`
	// A slack event is published if the user is not on the branch whitelist
	BranchWhiteList map[string][]string `json:"branch_whitelist,omitempty"`
}

// Welcome is config for the welcome plugin
type Welcome struct {
	// MessageTemplate is the welcome message template to post on new-contributor PRs
	// For the info struct see prow/plugins/welcome/welcome.go's PRInfo
	// TODO(bentheelder): make this be configurable per-repo?
	MessageTemplate string `json:"message_template,omitempty"`
}

// CherryPickUnapproved is the config for the cherrypick-unapproved plugin.
type CherryPickUnapproved struct {
	// BranchRegexp is the regular expression for branch names such that
	// the plugin treats only PRs against these branch names as cherrypick PRs.
	// Compiles into BranchRe during config load.
	BranchRegexp string         `json:"branchregexp,omitempty"`
	BranchRe     *regexp.Regexp `json:"-"`
	// Comment is the comment added by the plugin while adding the
	// `do-not-merge/cherry-pick-not-approved` label.
	Comment string `json:"comment,omitempty"`
}

// RequireMatchingLabel is the config for the require-matching-label plugin.
type RequireMatchingLabel struct {
	// Org is the GitHub organization that this config applies to.
	Org string `json:"org,omitempty"`
	// Repo is the GitHub repository within Org that this config applies to.
	// This fields may be omitted to apply this config across all repos in Org.
	Repo string `json:"repo,omitempty"`
	// Branch is the branch ref of PRs that this config applies to.
	// This field is only valid if `prs: true` and may be omitted to apply this
	// config across all branches in the repo or org.
	Branch string `json:"branch,omitempty"`
	// PRs is a bool indicating if this config applies to PRs.
	PRs bool `json:"prs,omitempty"`
	// Issues is a bool indicating if this config applies to issues.
	Issues bool `json:"issues,omitempty"`

	// Regexp is the string specifying the regular expression used to look for
	// matching labels.
	Regexp string `json:"regexp,omitempty"`
	// Re is the compiled version of Regexp. It should not be specified in config.
	Re *regexp.Regexp `json:"-"`

	// MissingLabel is the label to apply if an issue does not have any label
	// matching the Regexp.
	MissingLabel string `json:"missing_label,omitempty"`
	// MissingComment is the comment to post when we add the MissingLabel to an
	// issue. This is typically used to explain why MissingLabel was added and
	// how to move forward.
	// This field is optional. If unspecified, no comment is created when labeling.
	MissingComment string `json:"missing_comment,omitempty"`

	// GracePeriod is the amount of time to wait before processing newly opened
	// or reopened issues and PRs. This delay allows other automation to apply
	// labels before we look for matching labels.
	// Defaults to '5s'.
	GracePeriod         string        `json:"grace_period,omitempty"`
	GracePeriodDuration time.Duration `json:"-"`
}

// validate checks the following properties:
// - Org, Regexp, MissingLabel, and GracePeriod must be non-empty.
// - Repo does not contain a '/' (should use Org+Repo).
// - At least one of PRs or Issues must be true.
// - Branch only specified if 'prs: true'
// - MissingLabel must not match Regexp.
func (r RequireMatchingLabel) validate() error {
	if r.Org == "" {
		return errors.New("must specify 'org'")
	}
	if strings.Contains(r.Repo, "/") {
		return errors.New("'repo' may not contain '/'; specify the organization with 'org'")
	}
	if r.Regexp == "" {
		return errors.New("must specify 'regexp'")
	}
	if r.MissingLabel == "" {
		return errors.New("must specify 'missing_label'")
	}
	if r.GracePeriod == "" {
		return errors.New("must specify 'grace_period'")
	}
	if !r.PRs && !r.Issues {
		return errors.New("must specify 'prs: true' and/or 'issues: true'")
	}
	if !r.PRs && r.Branch != "" {
		return errors.New("branch cannot be specified without `prs: true'")
	}
	if r.Re.MatchString(r.MissingLabel) {
		return errors.New("'regexp' must not match 'missing_label'")
	}
	return nil
}

// Describe generates a human readable description of the behavior that this
// configuration specifies.
func (r RequireMatchingLabel) Describe() string {
	str := &strings.Builder{}
	fmt.Fprintf(str, "Applies the '%s' label ", r.MissingLabel)
	if r.MissingComment == "" {
		fmt.Fprint(str, "to ")
	} else {
		fmt.Fprint(str, "and comments on ")
	}

	if r.Issues {
		fmt.Fprint(str, "Issues ")
		if r.PRs {
			fmt.Fprint(str, "and ")
		}
	}
	if r.PRs {
		if r.Branch != "" {
			fmt.Fprintf(str, "'%s' branch ", r.Branch)
		}
		fmt.Fprint(str, "PRs ")
	}

	if r.Repo == "" {
		fmt.Fprintf(str, "in the '%s' GitHub org ", r.Org)
	} else {
		fmt.Fprintf(str, "in the '%s/%s' GitHub repo ", r.Org, r.Repo)
	}
	fmt.Fprintf(str, "that have no labels matching the regular expression '%s'.", r.Regexp)
	return str.String()
}

// TriggerFor finds the Trigger for a repo, if one exists
// a trigger can be listed for the repo itself or for the
// owning organization
func (c *Configuration) TriggerFor(org, repo string) *Trigger {
	for _, tr := range c.Triggers {
		for _, r := range tr.Repos {
			if r == org || r == fmt.Sprintf("%s/%s", org, repo) {
				return &tr
			}
		}
	}
	return nil
}

func (c *Configuration) setDefaults() {
	if len(c.ConfigUpdater.Maps) == 0 {
		cf := c.ConfigUpdater.ConfigFile
		if cf == "" {
			cf = "prow/config.yaml"
		} else {
			logrus.Warnf(`config_file is deprecated, please switch to "maps": {"%s": "config"} before July 2018`, cf)
		}
		pf := c.ConfigUpdater.PluginFile
		if pf == "" {
			pf = "prow/plugins.yaml"
		} else {
			logrus.Warnf(`plugin_file is deprecated, please switch to "maps": {"%s": "plugins"} before July 2018`, pf)
		}
		c.ConfigUpdater.Maps = map[string]ConfigMapSpec{
			cf: {
				Name: "config",
			},
			pf: {
				Name: "plugins",
			},
		}
	}
	for repo, plugins := range c.ExternalPlugins {
		for i, p := range plugins {
			if p.Endpoint != "" {
				continue
			}
			c.ExternalPlugins[repo][i].Endpoint = fmt.Sprintf("http://%s", p.Name)
		}
	}
	if c.Blunderbuss.ReviewerCount == nil && c.Blunderbuss.FileWeightCount == nil {
		c.Blunderbuss.ReviewerCount = new(int)
		*c.Blunderbuss.ReviewerCount = defaultBlunderbussReviewerCount
	}
	for i, trigger := range c.Triggers {
		if trigger.TrustedOrg == "" || trigger.JoinOrgURL != "" {
			continue
		}
		c.Triggers[i].JoinOrgURL = fmt.Sprintf("https://github.com/orgs/%s/people", trigger.TrustedOrg)
	}
	if c.SigMention.Regexp == "" {
		c.SigMention.Regexp = `(?m)@kubernetes/sig-([\w-]*)-(misc|test-failures|bugs|feature-requests|proposals|pr-reviews|api-reviews)`
	}
	if c.Owners.LabelsBlackList == nil {
		c.Owners.LabelsBlackList = []string{"approved", "lgtm"}
	}
	if c.CherryPickUnapproved.BranchRegexp == "" {
		c.CherryPickUnapproved.BranchRegexp = `^release-.*$`
	}
	if c.CherryPickUnapproved.Comment == "" {
		c.CherryPickUnapproved.Comment = `This PR is not for the master branch but does not have the ` + "`cherry-pick-approved`" + `  label. Adding the ` + "`do-not-merge/cherry-pick-not-approved`" + `  label.

To approve the cherry-pick, please assign the patch release manager for the release branch by writing ` + "`/assign @username`" + ` in a comment when ready.

The list of patch release managers for each release can be found [here](https://git.k8s.io/sig-release/release-managers.md).`
	}

	for i, rml := range c.RequireMatchingLabel {
		if rml.GracePeriod == "" {
			c.RequireMatchingLabel[i].GracePeriod = "5s"
		}
	}
}

// Load attempts to load config from the path. It returns an error if either
// the file can't be read or it contains an unknown plugin.
func (pa *PluginAgent) Load(path string) error {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	np := &Configuration{}
	if err := yaml.Unmarshal(b, np); err != nil {
		return err
	}

	if len(np.Plugins) == 0 {
		logrus.Warn("no plugins specified-- check syntax?")
	}

	// Defaulting should run before validation.
	np.setDefaults()
	// Regexp compilation should run after defaulting, but before validation.
	if err := compileRegexpsAndDurations(np); err != nil {
		return err
	}

	if err := validatePlugins(np.Plugins); err != nil {
		return err
	}
	if err := validateExternalPlugins(np.ExternalPlugins); err != nil {
		return err
	}
	if err := validateBlunderbuss(&np.Blunderbuss); err != nil {
		return err
	}
	if err := validateConfigUpdater(&np.ConfigUpdater); err != nil {
		return err
	}
	if err := validateSizes(np.Size); err != nil {
		return err
	}
	if err := validateRequireMatchingLabel(np.RequireMatchingLabel); err != nil {
		return err
	}
	pa.Set(np)
	return nil
}

func (pa *PluginAgent) Config() *Configuration {
	pa.mut.Lock()
	defer pa.mut.Unlock()
	return pa.configuration
}

// validatePlugins will return error if
// there are unknown or duplicated plugins.
func validatePlugins(plugins map[string][]string) error {
	var errors []string
	for _, configuration := range plugins {
		for _, plugin := range configuration {
			if _, ok := pluginHelp[plugin]; !ok {
				errors = append(errors, fmt.Sprintf("unknown plugin: %s", plugin))
			}
		}
	}
	for repo, repoConfig := range plugins {
		if strings.Contains(repo, "/") {
			org := strings.Split(repo, "/")[0]
			if dupes := findDuplicatedPluginConfig(repoConfig, plugins[org]); len(dupes) > 0 {
				errors = append(errors, fmt.Sprintf("plugins %v are duplicated for %s and %s", dupes, repo, org))
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("invalid plugin configuration:\n\t%v", strings.Join(errors, "\n\t"))
	}
	return nil
}

func validateSizes(size *Size) error {
	if size == nil {
		return nil
	}

	if size.S > size.M || size.M > size.L || size.L > size.Xl || size.Xl > size.Xxl {
		return errors.New("invalid size plugin configuration - one of the smaller sizes is bigger than a larger one")
	}

	return nil
}

func findDuplicatedPluginConfig(repoConfig, orgConfig []string) []string {
	var dupes []string
	for _, repoPlugin := range repoConfig {
		for _, orgPlugin := range orgConfig {
			if repoPlugin == orgPlugin {
				dupes = append(dupes, repoPlugin)
			}
		}
	}

	return dupes
}

func validateExternalPlugins(pluginMap map[string][]ExternalPlugin) error {
	var errors []string

	for repo, plugins := range pluginMap {
		if !strings.Contains(repo, "/") {
			continue
		}
		org := strings.Split(repo, "/")[0]

		var orgConfig []string
		for _, p := range pluginMap[org] {
			orgConfig = append(orgConfig, p.Name)
		}

		var repoConfig []string
		for _, p := range plugins {
			repoConfig = append(repoConfig, p.Name)
		}

		if dupes := findDuplicatedPluginConfig(repoConfig, orgConfig); len(dupes) > 0 {
			errors = append(errors, fmt.Sprintf("external plugins %v are duplicated for %s and %s", dupes, repo, org))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("invalid plugin configuration:\n\t%v", strings.Join(errors, "\n\t"))
	}
	return nil
}

func validateBlunderbuss(b *Blunderbuss) error {
	if b.ReviewerCount != nil && b.FileWeightCount != nil {
		return errors.New("cannot use both request_count and file_weight_count in blunderbuss")
	}
	if b.ReviewerCount != nil && *b.ReviewerCount < 1 {
		return fmt.Errorf("invalid request_count: %v (needs to be positive)", *b.ReviewerCount)
	}
	if b.FileWeightCount != nil && *b.FileWeightCount < 1 {
		return fmt.Errorf("invalid file_weight_count: %v (needs to be positive)", *b.FileWeightCount)
	}
	return nil
}

func validateConfigUpdater(updater *ConfigUpdater) error {
	files := sets.NewString()
	configMapKeys := map[string]sets.String{}
	for file, config := range updater.Maps {
		if files.Has(file) {
			return fmt.Errorf("file %s listed more than once in config updater config", file)
		}
		files.Insert(file)

		key := config.Key
		if key == "" {
			key = path.Base(file)
		}

		if _, ok := configMapKeys[config.Name]; ok {
			if configMapKeys[config.Name].Has(key) {
				return fmt.Errorf("key %s in configmap %s updated with more than one file", key, config.Name)
			}
			configMapKeys[config.Name].Insert(key)
		} else {
			configMapKeys[config.Name] = sets.NewString(key)
		}
	}
	return nil
}

func validateRequireMatchingLabel(rs []RequireMatchingLabel) error {
	for i, r := range rs {
		if err := r.validate(); err != nil {
			return fmt.Errorf("error validating require_matching_label config #%d: %v", i, err)
		}
	}
	return nil
}

func compileRegexpsAndDurations(pc *Configuration) error {
	cRe, err := regexp.Compile(pc.SigMention.Regexp)
	if err != nil {
		return err
	}
	pc.SigMention.Re = cRe

	branchRe, err := regexp.Compile(pc.CherryPickUnapproved.BranchRegexp)
	if err != nil {
		return err
	}
	pc.CherryPickUnapproved.BranchRe = branchRe

	rs := pc.RequireMatchingLabel
	for i := range rs {
		re, err := regexp.Compile(rs[i].Regexp)
		if err != nil {
			return fmt.Errorf("failed to compile label regexp: %q, error: %v", rs[i].Regexp, err)
		}
		rs[i].Re = re

		var dur time.Duration
		dur, err = time.ParseDuration(rs[i].GracePeriod)
		if err != nil {
			return fmt.Errorf("failed to compile grace period duration: %q, error: %v", rs[i].GracePeriod, err)
		}
		rs[i].GracePeriodDuration = dur
	}
	return nil
}

// Set attempts to set the plugins that are enabled on repos. Plugins are listed
// as a map from repositories to the list of plugins that are enabled on them.
// Specifying simply an org name will also work, and will enable the plugin on
// all repos in the org.
func (pa *PluginAgent) Set(pc *Configuration) {
	pa.mut.Lock()
	defer pa.mut.Unlock()
	pa.configuration = pc
}

// Start starts polling path for plugin config. If the first attempt fails,
// then start returns the error. Future errors will halt updates but not stop.
func (pa *PluginAgent) Start(path string) error {
	if err := pa.Load(path); err != nil {
		return err
	}
	ticker := time.Tick(1 * time.Minute)
	go func() {
		for range ticker {
			if err := pa.Load(path); err != nil {
				logrus.WithField("path", path).WithError(err).Error("Error loading plugin config.")
			}
		}
	}()
	return nil
}

// GenericCommentHandlers returns a map of plugin names to handlers for the repo.
func (pa *PluginAgent) GenericCommentHandlers(owner, repo string) map[string]GenericCommentHandler {
	pa.mut.Lock()
	defer pa.mut.Unlock()

	hs := map[string]GenericCommentHandler{}
	for _, p := range pa.getPlugins(owner, repo) {
		if h, ok := genericCommentHandlers[p]; ok {
			hs[p] = h
		}
	}
	return hs
}

// IssueHandlers returns a map of plugin names to handlers for the repo.
func (pa *PluginAgent) IssueHandlers(owner, repo string) map[string]IssueHandler {
	pa.mut.Lock()
	defer pa.mut.Unlock()

	hs := map[string]IssueHandler{}
	for _, p := range pa.getPlugins(owner, repo) {
		if h, ok := issueHandlers[p]; ok {
			hs[p] = h
		}
	}
	return hs
}

// IssueCommentHandlers returns a map of plugin names to handlers for the repo.
func (pa *PluginAgent) IssueCommentHandlers(owner, repo string) map[string]IssueCommentHandler {
	pa.mut.Lock()
	defer pa.mut.Unlock()

	hs := map[string]IssueCommentHandler{}
	for _, p := range pa.getPlugins(owner, repo) {
		if h, ok := issueCommentHandlers[p]; ok {
			hs[p] = h
		}
	}

	return hs
}

// PullRequestHandlers returns a map of plugin names to handlers for the repo.
func (pa *PluginAgent) PullRequestHandlers(owner, repo string) map[string]PullRequestHandler {
	pa.mut.Lock()
	defer pa.mut.Unlock()

	hs := map[string]PullRequestHandler{}
	for _, p := range pa.getPlugins(owner, repo) {
		if h, ok := pullRequestHandlers[p]; ok {
			hs[p] = h
		}
	}

	return hs
}

// ReviewEventHandlers returns a map of plugin names to handlers for the repo.
func (pa *PluginAgent) ReviewEventHandlers(owner, repo string) map[string]ReviewEventHandler {
	pa.mut.Lock()
	defer pa.mut.Unlock()

	hs := map[string]ReviewEventHandler{}
	for _, p := range pa.getPlugins(owner, repo) {
		if h, ok := reviewEventHandlers[p]; ok {
			hs[p] = h
		}
	}

	return hs
}

// ReviewCommentEventHandlers returns a map of plugin names to handlers for the repo.
func (pa *PluginAgent) ReviewCommentEventHandlers(owner, repo string) map[string]ReviewCommentEventHandler {
	pa.mut.Lock()
	defer pa.mut.Unlock()

	hs := map[string]ReviewCommentEventHandler{}
	for _, p := range pa.getPlugins(owner, repo) {
		if h, ok := reviewCommentEventHandlers[p]; ok {
			hs[p] = h
		}
	}

	return hs
}

// StatusEventHandlers returns a map of plugin names to handlers for the repo.
func (pa *PluginAgent) StatusEventHandlers(owner, repo string) map[string]StatusEventHandler {
	pa.mut.Lock()
	defer pa.mut.Unlock()

	hs := map[string]StatusEventHandler{}
	for _, p := range pa.getPlugins(owner, repo) {
		if h, ok := statusEventHandlers[p]; ok {
			hs[p] = h
		}
	}

	return hs
}

// PushEventHandlers returns a map of plugin names to handlers for the repo.
func (pa *PluginAgent) PushEventHandlers(owner, repo string) map[string]PushEventHandler {
	pa.mut.Lock()
	defer pa.mut.Unlock()

	hs := map[string]PushEventHandler{}
	for _, p := range pa.getPlugins(owner, repo) {
		if h, ok := pushEventHandlers[p]; ok {
			hs[p] = h
		}
	}

	return hs
}

// getPlugins returns a list of plugins that are enabled on a given (org, repository).
func (pa *PluginAgent) getPlugins(owner, repo string) []string {
	var plugins []string

	fullName := fmt.Sprintf("%s/%s", owner, repo)
	plugins = append(plugins, pa.configuration.Plugins[owner]...)
	plugins = append(plugins, pa.configuration.Plugins[fullName]...)

	return plugins
}

func EventsForPlugin(name string) []string {
	var events []string
	if _, ok := issueHandlers[name]; ok {
		events = append(events, "issue")
	}
	if _, ok := issueCommentHandlers[name]; ok {
		events = append(events, "issue_comment")
	}
	if _, ok := pullRequestHandlers[name]; ok {
		events = append(events, "pull_request")
	}
	if _, ok := pushEventHandlers[name]; ok {
		events = append(events, "push")
	}
	if _, ok := reviewEventHandlers[name]; ok {
		events = append(events, "pull_request_review")
	}
	if _, ok := reviewCommentEventHandlers[name]; ok {
		events = append(events, "pull_request_review_comment")
	}
	if _, ok := statusEventHandlers[name]; ok {
		events = append(events, "status")
	}
	if _, ok := genericCommentHandlers[name]; ok {
		events = append(events, "GenericCommentEvent (any event for user text)")
	}
	return events
}

func (c *Configuration) EnabledReposForPlugin(plugin string) (orgs, repos []string) {
	for repo, plugins := range c.Plugins {
		found := false
		for _, candidate := range plugins {
			if candidate == plugin {
				found = true
				break
			}
		}
		if found {
			if strings.Contains(repo, "/") {
				repos = append(repos, repo)
			} else {
				orgs = append(orgs, repo)
			}
		}
	}
	return
}

func (c *Configuration) EnabledReposForExternalPlugin(plugin string) (orgs, repos []string) {
	for repo, plugins := range c.ExternalPlugins {
		found := false
		for _, candidate := range plugins {
			if candidate.Name == plugin {
				found = true
				break
			}
		}
		if found {
			if strings.Contains(repo, "/") {
				repos = append(repos, repo)
			} else {
				orgs = append(orgs, repo)
			}
		}
	}
	return
}
