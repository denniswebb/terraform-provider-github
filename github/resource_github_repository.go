package github

import (
	"context"
	"fmt"
	"log"

	"github.com/google/go-github/github"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceGithubRepository() *schema.Resource {

	return &schema.Resource{
		Create: resourceGithubRepositoryCreate,
		Read:   resourceGithubRepositoryRead,
		Update: resourceGithubRepositoryUpdate,
		Delete: resourceGithubRepositoryDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"homepage_url": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"private": {
				Type:     schema.TypeBool,
				Optional: true,
			},
			"has_issues": {
				Type:     schema.TypeBool,
				Optional: true,
			},
			"has_wiki": {
				Type:     schema.TypeBool,
				Optional: true,
			},
			"allow_merge_commit": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"allow_squash_merge": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"allow_rebase_merge": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"has_downloads": {
				Type:     schema.TypeBool,
				Optional: true,
			},
			"auto_init": {
				Type:     schema.TypeBool,
				Optional: true,
			},
			"default_branch": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Can only be set after initial repository creation, and only if the target branch exists",
			},
			"license_template": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"gitignore_template": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"full_name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"ssh_clone_url": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"svn_url": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"git_clone_url": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"http_clone_url": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceGithubRepositoryObject(d *schema.ResourceData) *github.Repository {
	name := d.Get("name").(string)
	description := d.Get("description").(string)
	homepageUrl := d.Get("homepage_url").(string)
	private := d.Get("private").(bool)
	hasIssues := d.Get("has_issues").(bool)
	hasWiki := d.Get("has_wiki").(bool)
	allowMergeCommit := d.Get("allow_merge_commit").(bool)
	allowSquashMerge := d.Get("allow_squash_merge").(bool)
	allowRebaseMerge := d.Get("allow_rebase_merge").(bool)
	hasDownloads := d.Get("has_downloads").(bool)
	autoInit := d.Get("auto_init").(bool)
	licenseTemplate := d.Get("license_template").(string)
	gitIgnoreTemplate := d.Get("gitignore_template").(string)

	repo := &github.Repository{
		Name:              &name,
		Description:       &description,
		Homepage:          &homepageUrl,
		Private:           &private,
		HasIssues:         &hasIssues,
		HasWiki:           &hasWiki,
		AllowMergeCommit:  &allowMergeCommit,
		AllowSquashMerge:  &allowSquashMerge,
		AllowRebaseMerge:  &allowRebaseMerge,
		HasDownloads:      &hasDownloads,
		AutoInit:          &autoInit,
		LicenseTemplate:   &licenseTemplate,
		GitignoreTemplate: &gitIgnoreTemplate,
	}

	return repo
}

func resourceGithubRepositoryCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Organization).client

	if _, ok := d.GetOk("default_branch"); ok {
		return fmt.Errorf("Cannot set the default branch on a new repository.")
	}

	repoReq := resourceGithubRepositoryObject(d)
	log.Printf("[DEBUG] create github repository %s/%s", meta.(*Organization).name, *repoReq.Name)
	repo, _, err := client.Repositories.Create(context.TODO(), meta.(*Organization).name, repoReq)
	if err != nil {
		return err
	}
	d.SetId(*repo.Name)

	return resourceGithubRepositoryRead(d, meta)
}

func resourceGithubRepositoryRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Organization).client
	repoName := d.Id()

	log.Printf("[DEBUG] read github repository %s/%s", meta.(*Organization).name, repoName)
	repo, resp, err := client.Repositories.Get(context.TODO(), meta.(*Organization).name, repoName)
	if err != nil {
		if resp != nil && resp.StatusCode == 404 {
			log.Printf(
				"[WARN] removing %s/%s from state because it no longer exists in github",
				meta.(*Organization).name,
				repoName,
			)
			d.SetId("")
			return nil
		}
		return err
	}
	d.Set("name", repoName)
	d.Set("description", repo.Description)
	d.Set("homepage_url", repo.Homepage)
	d.Set("private", repo.Private)
	d.Set("has_issues", repo.HasIssues)
	d.Set("has_wiki", repo.HasWiki)
	d.Set("allow_merge_commit", repo.AllowMergeCommit)
	d.Set("allow_squash_merge", repo.AllowSquashMerge)
	d.Set("allow_rebase_merge", repo.AllowRebaseMerge)
	d.Set("has_downloads", repo.HasDownloads)
	d.Set("full_name", repo.FullName)
	d.Set("default_branch", repo.DefaultBranch)
	d.Set("ssh_clone_url", repo.SSHURL)
	d.Set("svn_url", repo.SVNURL)
	d.Set("git_clone_url", repo.GitURL)
	d.Set("http_clone_url", repo.CloneURL)
	return nil
}

func resourceGithubRepositoryUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Organization).client
	repoReq := resourceGithubRepositoryObject(d)
	// Can only set `default_branch` on an already created repository with the target branches ref already in-place
	if v, ok := d.GetOk("default_branch"); ok {
		branch := v.(string)
		// If branch is "master", and the repository hasn't been initialized yet, setting this value will fail
		if branch != "master" {
			repoReq.DefaultBranch = &branch
		}
	}

	repoName := d.Id()
	log.Printf("[DEBUG] update github repository %s/%s", meta.(*Organization).name, repoName)
	repo, _, err := client.Repositories.Edit(context.TODO(), meta.(*Organization).name, repoName, repoReq)
	if err != nil {
		return err
	}
	d.SetId(*repo.Name)

	return resourceGithubRepositoryRead(d, meta)
}

func resourceGithubRepositoryDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Organization).client
	repoName := d.Id()
	log.Printf("[DEBUG] delete github repository %s/%s", meta.(*Organization).name, repoName)
	_, err := client.Repositories.Delete(context.TODO(), meta.(*Organization).name, repoName)
	return err
}
