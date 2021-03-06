package internalversion

import (
	oauth "github.com/openshift/origin/pkg/oauth/apis/oauth"
	scheme "github.com/openshift/origin/pkg/oauth/generated/internalclientset/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// OAuthClientsGetter has a method to return a OAuthClientInterface.
// A group's client should implement this interface.
type OAuthClientsGetter interface {
	OAuthClients() OAuthClientInterface
}

// OAuthClientInterface has methods to work with OAuthClient resources.
type OAuthClientInterface interface {
	Create(*oauth.OAuthClient) (*oauth.OAuthClient, error)
	Update(*oauth.OAuthClient) (*oauth.OAuthClient, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*oauth.OAuthClient, error)
	List(opts v1.ListOptions) (*oauth.OAuthClientList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *oauth.OAuthClient, err error)
	OAuthClientExpansion
}

// oAuthClients implements OAuthClientInterface
type oAuthClients struct {
	client rest.Interface
}

// newOAuthClients returns a OAuthClients
func newOAuthClients(c *OauthClient) *oAuthClients {
	return &oAuthClients{
		client: c.RESTClient(),
	}
}

// Get takes name of the oAuthClient, and returns the corresponding oAuthClient object, and an error if there is any.
func (c *oAuthClients) Get(name string, options v1.GetOptions) (result *oauth.OAuthClient, err error) {
	result = &oauth.OAuthClient{}
	err = c.client.Get().
		Resource("oauthclients").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of OAuthClients that match those selectors.
func (c *oAuthClients) List(opts v1.ListOptions) (result *oauth.OAuthClientList, err error) {
	result = &oauth.OAuthClientList{}
	err = c.client.Get().
		Resource("oauthclients").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested oAuthClients.
func (c *oAuthClients) Watch(opts v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Resource("oauthclients").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a oAuthClient and creates it.  Returns the server's representation of the oAuthClient, and an error, if there is any.
func (c *oAuthClients) Create(oAuthClient *oauth.OAuthClient) (result *oauth.OAuthClient, err error) {
	result = &oauth.OAuthClient{}
	err = c.client.Post().
		Resource("oauthclients").
		Body(oAuthClient).
		Do().
		Into(result)
	return
}

// Update takes the representation of a oAuthClient and updates it. Returns the server's representation of the oAuthClient, and an error, if there is any.
func (c *oAuthClients) Update(oAuthClient *oauth.OAuthClient) (result *oauth.OAuthClient, err error) {
	result = &oauth.OAuthClient{}
	err = c.client.Put().
		Resource("oauthclients").
		Name(oAuthClient.Name).
		Body(oAuthClient).
		Do().
		Into(result)
	return
}

// Delete takes name of the oAuthClient and deletes it. Returns an error if one occurs.
func (c *oAuthClients) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Resource("oauthclients").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *oAuthClients) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Resource("oauthclients").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched oAuthClient.
func (c *oAuthClients) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *oauth.OAuthClient, err error) {
	result = &oauth.OAuthClient{}
	err = c.client.Patch(pt).
		Resource("oauthclients").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
