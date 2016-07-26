package plugins

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"

	"github.com/InnovaCo/serve/manifest"
	"github.com/InnovaCo/serve/utils/gabs"
)

func init() {
	manifest.PluginRegestry.Add("gocd.pipeline.create", goCdPipelineCreate{})
}

/**
 * plugin for manifest section "goCd.pipeline.create"
 * section structure:
 *
 * goCd.pipeline.create:
 *   api-url: goCd_URL
 *   environment: ENV
 *   branch: BRANCH
 *   allowed-branches: [BRANCH, ...]
 *   pipeline:
 *     group: GROUP
 *     pipeline:
 *       according to the description: https://api.go.cd/current/#the-pipeline-config-object
 */

type goCdCredents struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type goCdPipelineCreate struct{}

func (p goCdPipelineCreate) Run(data manifest.Manifest) error {
	name := data.GetString("pipeline.pipeline.name")
	url := data.GetString("api-url")
	body := data.GetTree("pipeline").String()
	branch := data.GetString("branch")

	m := false
	for _, b := range data.GetArray("allowed-branches") {
		re := b.Unwrap().(string)
		if re == "*" || re == branch {
			m = true
			break
		} else if m, _ = regexp.MatchString(re, branch); m {
			break
		}
	}

	if !m {
		log.Println("branch ", branch, " not in ", data.GetString("allowed-branches"))
		return nil
	}

	resp, err := goCdRequest("GET", url+"/go/api/admin/pipelines/"+name, "",
		map[string]string{"Accept": "application/vnd.go.cd.v2+json"})
	if err != nil {
		return err
	}

	if resp.StatusCode == http.StatusOK {
		err = goCdUpdate(name, data.GetString("environment"), url, body,
			map[string]string{"If-Match": resp.Header.Get("ETag"), "Accept": "application/vnd.go.cd.v2+json"})
	} else if resp.StatusCode == http.StatusNotFound {
		err = goCdCreate(name, data.GetString("environment"), url, body,
			map[string]string{"Accept": "application/vnd.go.cd.v2+json"})
	} else {
		return fmt.Errorf("Operation error: %s", resp.Status)
	}

	if err != nil {
		return err
	}

	return nil
}

func goCdCreate(name string, env string, resource string, body string, headers map[string]string) error {
	if resp, err := goCdRequest("POST", resource+"/go/api/admin/pipelines", body, headers); err != nil {
		return err
	} else if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Operation error: %s", resp.Status)
	}
	data, tag, err := goCdChangeEnv(resource, env, name, "")
	if err != nil {
		return err
	}

	if resp, err := goCdRequest("PUT", resource+"/go/api/admin/environments/"+env, data,
		map[string]string{"If-Match": tag, "Accept": "application/vnd.go.cd.v1+json"}); err != nil {
		return err
	} else if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Operation error: %s", resp.Status)
	}

	if resp, err := goCdRequest("POST", resource+"/go/api/pipelines/"+name+"/unpause", "",
		map[string]string{"Confirm": "true"}); err != nil {
		return err
	} else if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Operation error: %s", resp.Status)
	}

	return nil
}

func goCdUpdate(name string, env string, resource string, body string, headers map[string]string) error {
	fmt.Println(env)

	if resp, err := goCdRequest("PUT", resource+"/go/api/admin/pipelines/"+name, body, headers); err != nil {
		return err
	} else if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Operation error: %s", resp.Status)
	}

	if cEnv, err := goCdFindEnv(resource, name); err == nil {
		if env != cEnv && cEnv != "" {

			data, tag, err := goCdChangeEnv(resource, cEnv, "", name)
			if err != nil {
				return err
			}
			if resp, err := goCdRequest("PUT", resource+"/go/api/admin/environments/"+cEnv, data,
				map[string]string{"If-Match": tag, "Accept": "application/vnd.go.cd.v1+json"}); err != nil {
				return err
			} else if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("Operation error: %s", resp.Status)
			}
		}

		data, tag, err := goCdChangeEnv(resource, env, name, "")
		if err != nil {
			return err
		}

		if resp, err := goCdRequest("PUT", resource+"/go/api/admin/environments/"+env, data,
			map[string]string{"If-Match": tag, "Accept": "application/vnd.go.cd.v1+json"}); err != nil {
			return err
		} else if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("Operation error: %s", resp.Status)
		}
	} else {
		return err
	}

	if resp, err := goCdRequest("POST", resource+"/go/api/pipelines/"+name+"/unpause", "",
		map[string]string{"Confirm": "true"}); err != nil {
		return err
	} else if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Operation error: %s", resp.Status)
	}

	return nil
}

func goCdDelete(name string, env string, resource string, headers map[string]string) error {
	data, tag, err := goCdChangeEnv(resource, env, "", name)
	if err != nil {
		return err
	}

	log.Println(data)

	if resp, err := goCdRequest("PUT", resource+"/go/api/admin/environments/"+env, data,
		map[string]string{"If-Match": tag, "Accept": "application/vnd.go.cd.v1+json"}); err != nil {
		return err
	} else if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Operation error: %s", resp.Status)
	}

	if resp, err := goCdRequest("DELETE", resource+"/go/api/admin/pipelines/"+name, "", headers); err != nil {
		return err
	} else if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Operation error: %s", resp.Status)
	}

	return nil
}

func goCdChangeEnv(resource string, env string, addPipeline string, delPipeline string) (string, string, error) {
	log.Printf("change environment: %s", env)
	resp, err := goCdRequest("GET", resource+"/go/api/admin/environments/"+env, "",
		map[string]string{"Accept": "application/vnd.go.cd.v1+json"})
	if err != nil {
		return "", "", err
	} else if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("Operation error: %s", resp.Status)
	}

	data, err := ChangeJSON(resp, addPipeline, delPipeline)
	if err != nil {
		return "", "", err
	}

	return data, resp.Header.Get("ETag"), nil
}

func goCdFindEnv(resource string, pipeline string) (string, error) {
	resp, err := goCdRequest("GET", resource+"/go/api/admin/environments", "",
		map[string]string{"Accept": "application/vnd.go.cd.v1+json"})
	if err != nil {
		return "", err
	} else if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Operation error: %s", resp.Status)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	tree, err := gabs.ParseJSON(body)
	if err != nil {
		return "", err
	}

	envs, _ := tree.Path("_embedded.environments").Children()
	for _, env := range envs {
		envName := env.Path("name").Data().(string)
		pipelines, _ := env.Path("pipelines").Children()
		for _, pline := range pipelines {
			if pline.Path("name").Data().(string) == pipeline {
				return envName, nil
			}
		}
	}

	return "", nil
}

func goCdRequest(method string, resource string, body string, headers map[string]string) (*http.Response, error) {
	req, _ := http.NewRequest(method, resource, bytes.NewReader([]byte(body)))

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	//req.Header.Set("Accept", "application/vnd.go.cd.v1+json")
	req.Header.Set("Content-Type", "application/json")

	data, err := ioutil.ReadFile("/etc/serve/gocd_credentials")
	if err != nil {
		return nil, fmt.Errorf("Credentias file error: %v", err)
	}

	creds := &goCdCredents{}
	json.Unmarshal(data, creds)

	req.SetBasicAuth(creds.Login, creds.Password)

	log.Printf(" --> %s %s:\n%s\n%s\n", method, resource, req.Header, []byte(body)[:512])

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	} else {
		log.Printf("<-- %s", resp.Status)
	}

	return resp, nil
}

func ChangeJSON(resp *http.Response, addPipeline string, delPipeline string) (string, error) {
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return "", fmt.Errorf("read body error: %s", body)
	}

	tree, err := gabs.ParseJSON(body)
	if err != nil {
		return "", fmt.Errorf("parse body error: %s", body)
	}
	result := gabs.New()

	result.Set(tree.Path("name").Data(), "name")

	children, _ := tree.S("pipelines").Children()
	vals := []map[string]string{}
	for _, m := range children {
		name := m.Path("name").Data().(string)
		if (delPipeline != "") && (name == delPipeline) {
			continue
		}
		if (addPipeline != "") && (name == addPipeline) {
			addPipeline = ""
		}
		vals = append(vals, map[string]string{"name": name})
	}
	if addPipeline != "" {
		vals = append(vals, map[string]string{"name": addPipeline})
	}
	result.Set(vals, "pipelines")

	children, _ = tree.S("agents").Children()
	vals = []map[string]string{}
	for _, m := range children {
		vals = append(vals, map[string]string{"uuid": m.Path("uuid").Data().(string)})
	}
	result.Set(vals, "agents")
	result.Set(tree.Path("environment_variables").Data(), "environment_variables")

	return result.String(), nil
}
