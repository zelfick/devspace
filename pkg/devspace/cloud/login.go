package cloud

import (
	"context"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/config"
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/token"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/survey"
	"github.com/pkg/errors"
	"github.com/skratchdot/open-golang/open"
)

// LoginEndpoint is the cloud endpoint that will log you in
const LoginEndpoint = "/login?cli=true"

// LoginSuccessEndpoint is the url redirected to after successful login
const LoginSuccessEndpoint = "/login-success"

// TokenEndpoint is the endpoint where to get a token from
const TokenEndpoint = "/auth/token"

// GetToken returns a valid access token to the provider
func (p *Provider) GetToken() (string, error) {
	if p.Key == "" {
		return "", errors.New("Provider has no key specified")
	}
	if p.Token != "" && token.IsTokenValid(p.Token) {
		return p.Token, nil
	}

	resp, err := http.Get(p.Host + TokenEndpoint + "?key=" + p.Key)
	if err != nil {
		return "", errors.Wrap(err, "token request")
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", errors.Wrap(err, "read request body")
	}

	if resp.StatusCode != http.StatusOK {
		return "", errors.Errorf("Error retrieving token: Code %v => %s. Try to relogin with 'devspace login'", resp.StatusCode, string(body))
	}

	p.Token = string(body)
	if token.IsTokenValid(p.Token) == false {
		return "", errors.New("Received invalid token from provider")
	}

	err = p.Save()
	if err != nil {
		return "", errors.Wrap(err, "token save")
	}

	return p.Token, nil
}

// ReLogin loggs the user in with the given key or via browser
func ReLogin(providerConfig *latest.Config, cloudProvider string, key *string, log log.Logger) error {
	// Let's check if we are logged in first
	p := config.GetProvider(providerConfig, cloudProvider)
	if p == nil {
		cloudProviders := ""
		for _, p := range providerConfig.Providers {
			cloudProviders += p.Name + " "
		}

		return errors.Errorf("Cloud provider not found! Did you run `devspace add provider [url]`? Existing cloud providers: %s", cloudProviders)
	}

	provider := &Provider{
		*p,
		log,
	}
	if key != nil {
		provider.Token = ""
		provider.Key = *key

		// Check if we got access
		_, err := provider.GetSpaces()
		if err != nil {
			return errors.Errorf("Access denied for key %s: %v", *key, err)
		}
	} else {
		provider.Token = ""
		provider.Key = ""

		err := provider.Login()
		if err != nil {
			return errors.Wrap(err, "Login")
		}
	}

	log.Donef("Successfully logged into %s", provider.Name)

	// Login into registries
	err := provider.LoginIntoRegistries()
	if err != nil {
		log.Warnf("Error logging into docker registries: %v", err)
	}

	// Save config
	err = provider.Save()
	if err != nil {
		return err
	}

	return nil
}

// EnsureLoggedIn checks if the user is logged into a certain cloud provider and if not loggs the user in
func EnsureLoggedIn(providerConfig *latest.Config, cloudProvider string, log log.Logger) error {
	// Let's check if we are logged in first
	p := config.GetProvider(providerConfig, cloudProvider)
	if p == nil {
		cloudProviders := ""
		for _, p := range providerConfig.Providers {
			cloudProviders += p.Name + " "
		}

		return errors.Errorf("Cloud provider not found! Did you run `devspace add provider [url]`? Existing cloud providers: %s", cloudProviders)
	}

	provider := &Provider{
		*p,
		log,
	}
	if provider.Key == "" {
		provider.Token = ""

		err := provider.Login()
		if err != nil {
			return errors.Wrap(err, "ensure logged in")
		}

		log.Donef("Successfully logged into %s", provider.Name)

		// Login into registries
		err = provider.LoginIntoRegistries()
		if err != nil {
			log.Warnf("Error logging into docker registries: %v", err)
		}

		err = provider.Save()
		if err != nil {
			return err
		}
	}

	return nil
}

// Login logs the user into DevSpace Cloud
func (p *Provider) Login() error {
	var (
		url        = p.Host + LoginEndpoint
		ctx        = context.Background()
		keyChannel = make(chan string)
	)
	var key string

	server := startServer(p.Host+LoginSuccessEndpoint, keyChannel, p.Log)
	err := open.Run(url)
	if err != nil {
		p.Log.Infof("Unable to open web browser for login page.\n\n Please follow these instructions for manually loggin in:\n\n  1. Open this URL in a browser: %s\n  2. After logging in, click the 'Create Key' button\n  3. Enter a key name (e.g. my-key) and click 'Create Access Key'\n  4. Copy the generated key from the input field", p.Host+"/settings/access-keys")

		key, err = survey.Question(&survey.QuestionOptions{
			Question:   "5. Enter the access key here:",
			IsPassword: true,
		}, p.Log)
		if err != nil {
			return err
		}

		key = strings.TrimSpace(key)

		p.Log.WriteString("\n")

		providerConfig, err := config.ParseProviderConfig()
		if err != nil {
			return err
		}

		err = ReLogin(providerConfig, p.Name, &key, p.Log)
		if err != nil {
			return errors.Wrap(err, "login")
		}
	} else {
		p.Log.Infof("If the browser does not open automatically, please navigate to %s", url)
		p.Log.StartWait("Logging into cloud provider...")
		defer p.Log.StopWait()

		key = <-keyChannel
	}

	close(keyChannel)
	err = server.Shutdown(ctx)
	if err != nil {
		return err
	}

	p.Key = key
	return nil
}

func startServer(redirectURI string, keyChannel chan string, log log.Logger) *http.Server {
	srv := &http.Server{Addr: ":25853"}

	http.HandleFunc("/key", func(w http.ResponseWriter, r *http.Request) {
		keys, ok := r.URL.Query()["key"]
		if !ok || len(keys[0]) < 1 {
			log.Warn("Bad request")
		}

		keyChannel <- keys[0]
		http.Redirect(w, r, redirectURI, http.StatusSeeOther)
	})

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			// cannot panic, because this probably is an intentional close
		}
	}()

	// returning reference so caller can call Shutdown()
	return srv
}
