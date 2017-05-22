package bitbucket

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/hashicorp/terraform/helper/schema"
)

type CloneUrl struct {
	Href string `json:"href,omitempty"`
	Name string `json:"name,omitempty"`
}

type Repository struct {
	SCM         string `json:"scm,omitempty"`
	HasWiki     bool   `json:"has_wiki,omitempty"`
	HasIssues   bool   `json:"has_issues,omitempty"`
	Website     string `json:"website,omitempty"`
	IsPrivate   bool   `json:"is_private,omitempty"`
	ForkPolicy  string `json:"fork_policy,omitempty"`
	Language    string `json:"language,omitempty"`
	Description string `json:"description,omitempty"`
	Name        string `json:"name,omitempty"`
	UUID        string `json:"uuid,omitempty"`
	Project     struct {
		Key string `json:"key,omitempty"`
	} `json:"project,omitempty"`
	Links struct {
		Clone []CloneUrl `json:"clone,omitempty"`
	} `json:"links,omitempty"`
}

type PipelinesConfig struct {
	Pipelines bool `json:"enabled"`
}

type PipelinesVariable struct {
	Key     string `json:"key,omitempty"`
	Value   string `json:"value,omitempty"`
	Secured bool   `json:"secured"`
	// UUID    string `json:"uuid,omitempty"`
}

func resourceRepository() *schema.Resource {
	return &schema.Resource{
		Create: resourceRepositoryCreate,
		Update: resourceRepositoryUpdate,
		Read:   resourceRepositoryRead,
		Delete: resourceRepositoryDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"scm": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "git",
			},
			"has_wiki": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"has_issues": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"website": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"clone_ssh": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"clone_https": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"project_key": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"is_private": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"fork_policy": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "allow_forks",
			},
			"language": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"owner": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"pipelines": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"pipelines_variable": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"key": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"value": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"secured": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
						// "uuid": &schema.Schema{
						//	Type:     schema.TypeString,
						//	Computed: true,
						//},
					},
				},
			},
		},
	}
}

func newRepositoryFromResource(d *schema.ResourceData) *Repository {
	repo := &Repository{
		Name:        d.Get("name").(string),
		Language:    d.Get("language").(string),
		IsPrivate:   d.Get("is_private").(bool),
		Description: d.Get("description").(string),
		ForkPolicy:  d.Get("fork_policy").(string),
		HasWiki:     d.Get("has_wiki").(bool),
		HasIssues:   d.Get("has_issues").(bool),
		SCM:         d.Get("scm").(string),
		Website:     d.Get("website").(string),
	}

	repo.Project.Key = d.Get("project_key").(string)
	return repo
}

func enablePipelinesFromResource(d *schema.ResourceData) *PipelinesConfig {
	//return pipelines
	pipeline := &PipelinesConfig{
		Pipelines: d.Get("pipelines").(bool),
	}
	return pipeline
}

func newPipelinesVarFromResource(d *schema.ResourceData) *PipelinesVariable {
	pipelines_var := &PipelinesVariable{
		Key:     d.Get("key").(string),
		Value:   d.Get("value").(string),
		Secured: d.Get("secured").(bool),
	}
	return pipelines_var
}

func resourceRepositoryUpdate(d *schema.ResourceData, m interface{}) error {
	client := m.(*BitbucketClient)
	repository := newRepositoryFromResource(d)

	var jsonbuffer []byte

	jsonpayload := bytes.NewBuffer(jsonbuffer)
	enc := json.NewEncoder(jsonpayload)
	enc.Encode(repository)

	_, err := client.Put(fmt.Sprintf("2.0/repositories/%s/%s",
		d.Get("owner").(string),
		d.Get("name").(string),
	), jsonpayload)

	if err != nil {
		return err
	}

	pipelines := enablePipelinesFromResource(d)
	pipelinesbytedata, err := json.Marshal(pipelines)

	// do a put against pipelines endpoint
	_, err = client.Put(fmt.Sprintf("2.0/repositories/%s/%s/pipelines_config",
		d.Get("owner").(string),
		d.Get("name").(string),
	), bytes.NewBuffer(pipelinesbytedata))

	if err != nil {
		return err
	}

	// pipelines_var := newPipelinesVarFromResource(d)
	//pipelinesvarbytedata, err := json.Marshal(pipelines_var)

	// todo: get uuid of variable

	// do a put against variables endpoint
	// _, err = client.Put(fmt.Sprintf("2.0/repositories/%s/%s/pipelines_config/variables/%s",
	//	d.Get("owner").(string),
	//	d.Get("name").(string),
	// d.Get("uuid").(string),
	//), bytes.NewBuffer(pipelinesvarbytedata))

	//if err != nil {
	//	return err
	//}

	return resourceRepositoryRead(d, m)
}

func resourceRepositoryCreate(d *schema.ResourceData, m interface{}) error {
	client := m.(*BitbucketClient)
	repo := newRepositoryFromResource(d)

	bytedata, err := json.Marshal(repo)

	if err != nil {
		return err
	}

	_, err = client.Post(fmt.Sprintf("2.0/repositories/%s/%s",
		d.Get("owner").(string),
		d.Get("name").(string),
	), bytes.NewBuffer(bytedata))

	if err != nil {
		return err
	}

	d.SetId(string(fmt.Sprintf("%s/%s", d.Get("owner").(string), d.Get("name").(string))))

	if d.Get("pipelines") != nil {
		pipelines := enablePipelinesFromResource(d)
		pipelinesbytedata, err := json.Marshal(pipelines)

		_, err = client.Put(fmt.Sprintf("2.0/repositories/%s/%s/pipelines_config",
			d.Get("owner").(string),
			d.Get("name").(string),
		), bytes.NewBuffer(pipelinesbytedata))

		if err != nil {
			return err
		}
	}

	//fmt.Printf("%+v\n", pvar)
	// if d.Get("pipelines_variable") != nil {
	//	pipelines_var := newPipelinesVarFromResource(d)
	//	pipelinesvarbytedata, err := json.Marshal(pipelines_var)

	//	_, err = client.Put(fmt.Sprintf("2.0/repositories/%s/%s/pipelines_config/variables",
	//		d.Get("owner").(string),
	//		d.Get("name").(string),
	//	), bytes.NewBuffer(pipelinesvarbytedata))

	//	if err != nil {
	//		return err
	//	}
	//}

	return resourceRepositoryRead(d, m)
}

func resourceRepositoryRead(d *schema.ResourceData, m interface{}) error {

	client := m.(*BitbucketClient)
	repo_req, _ := client.Get(fmt.Sprintf("2.0/repositories/%s/%s",
		d.Get("owner").(string),
		d.Get("name").(string),
	))

	if repo_req.StatusCode == 200 {

		var repo Repository

		body, readerr := ioutil.ReadAll(repo_req.Body)
		if readerr != nil {
			return readerr
		}

		decodeerr := json.Unmarshal(body, &repo)
		if decodeerr != nil {
			return decodeerr
		}

		d.Set("scm", repo.SCM)
		d.Set("is_private", repo.IsPrivate)
		d.Set("has_wiki", repo.HasWiki)
		d.Set("has_issues", repo.HasIssues)
		d.Set("name", repo.Name)
		d.Set("language", repo.Language)
		d.Set("fork_policy", repo.ForkPolicy)
		d.Set("website", repo.Website)
		d.Set("description", repo.Description)
		d.Set("project_key", repo.Project.Key)

		for _, clone_url := range repo.Links.Clone {
			if clone_url.Name == "https" {
				d.Set("clone_https", clone_url.Href)
			} else {
				d.Set("clone_ssh", clone_url.Href)
			}
		}
	}

	// do a get against api to get pipelines config
	pipelines_req, _ := client.Get(fmt.Sprintf("2.0/repositories/%s/%s/pipelines_config",
		d.Get("owner").(string),
		d.Get("name").(string),
	))

	// unmarshal it into my struct
	if pipelines_req.StatusCode == 200 {

		var pipeline PipelinesConfig

		body, readerr := ioutil.ReadAll(pipelines_req.Body)
		if readerr != nil {
			return readerr
		}

		decodeerr := json.Unmarshal(body, &pipeline)
		if decodeerr != nil {
			return decodeerr
		}

		// set the resource data
		d.Set("pipelines", pipeline.Pipelines)
	}

	// GET for pipelines variables
	pipelines_var_req, _ := client.Get(
		fmt.Sprintf("2.0/repositories/%s/%s/pipelines_config/variables/",
			d.Get("owner").(string),
			d.Get("name").(string),
		))

	// unmarshal it into my struct
	if pipelines_var_req.StatusCode == 200 {

		var pipeline_var PipelinesVariable
		body, readerr := ioutil.ReadAll(pipelines_var_req.Body)
		if readerr != nil {
			return readerr
		}

		decodeerr := json.Unmarshal(body, &pipeline_var)
		if decodeerr != nil {
			return decodeerr
		}

		//fmt.Printf("%T\n", pipeline_var)
		// set the resource data
		d.Set("pipelines_variable.key", pipeline_var.Key)
		d.Set("value", pipeline_var.Value)
		d.Set("secured", pipeline_var.Secured)
		// d.Set("uuid", pipeline_var.UUID)
	}
	return nil
}

func resourceRepositoryDelete(d *schema.ResourceData, m interface{}) error {
	client := m.(*BitbucketClient)
	_, err := client.Delete(
		fmt.Sprintf("2.0/repositories/%s/%s",
			d.Get("owner").(string),
			d.Get("name").(string),
		))

	// DELETE for pipelines variables
	//pipelines_var_req, _ := client.Delete(
	//	fmt.Sprintf("/1.0/repositories/%s/%s/pipelines_config/variables/%s",
	//		d.Get("owner").(string),
	//		d.Get("name").(string),
	//		d.Get("variable_uuid").(string),
	//	))

	return err
}
