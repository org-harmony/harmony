// Package web provides a web server implementation for HARMONY and declaring its interface.
// Also, the web package handles utility and required web functionality to allow domain packages
// to easily extend upon this package and allow web communication.
package web

import (
	"context"
	"errors"
	"fmt"
	"github.com/org-harmony/harmony/src/core/hctx"
	"github.com/org-harmony/harmony/src/core/util"
	"github.com/org-harmony/harmony/src/core/validation"
	"html/template"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
)

// Pkg is the package name used for logging.
const Pkg = "sys.web"

var (
	// ErrNotPointerToStruct is returned when the input is not a pointer to a struct.
	ErrNotPointerToStruct = errors.New("input is not a pointer to a struct")
	// ErrUnexpectedReflection is returned when an unexpected reflection error occurs.
	ErrUnexpectedReflection = errors.New("unexpected reflection error")
	// ErrInternalReadForm is returned when an internal error occurs while reading the form.
	ErrInternalReadForm = errors.New("internal error reading form")
	// ErrInternal can be used to wrap unexpected internal errors whose message should not be displayed to the user.
	// In most cases instead of using ErrInternal, a more specific error should be used.
	ErrInternal = errors.New("harmony.error.generic-reload")
)

// Cfg is the config for the web package.
// It contains the config for the web server and the config for the UI.
type Cfg struct {
	Server *ServerCfg `toml:"server" hvalidate:"required"`
	UI     *UICfg     `toml:"ui" hvalidate:"required"`
}

// ServerCfg is the config for the web server. It contains the address and port to listen on and the base url.
// It also specifies the config for the asset file server.
type ServerCfg struct {
	AssetFsCfg *FileServerCfg `toml:"asset_fs" hvalidate:"required"`
	Addr       string         `toml:"address" env:"ADDR"`
	Port       string         `toml:"port" env:"PORT" hvalidate:"required"`
	BaseURL    string         `toml:"base_url" env:"BASE_URL" hvalidate:"required"`
}

// FileServerCfg is the config for a file server. It contains the root directory to assets and the route to serve them on.
type FileServerCfg struct {
	Root  string `toml:"root" hvalidate:"required"`
	Route string `toml:"route" hvalidate:"required"`
}

// Ctx is the web context. It is passed to the controller's handler function.
// It contains the router, config, templater store, navigation and template data extensions.
type Ctx struct {
	Router         Router
	Config         *Cfg
	TemplaterStore TemplaterStore
	Navigation     *Navigation
	Extensions     *TemplateDataExtensions
}

// Controller is convenience struct for handling web requests.
// The Controller is aware of the application context and the web context.
// The Controller implements the http.Handler interface and can therefore be used as a handler.
type Controller struct {
	appCtx  *hctx.AppCtx
	webCtx  *Ctx
	handler func(io IO) error
}

// HIO is a web.IO allowing for simplified access to the http.ResponseWriter and http.Request.
type HIO struct {
	writer   http.ResponseWriter
	request  *http.Request
	appCtx   *hctx.AppCtx
	webCtx   *Ctx
	baseData *BaseTemplateData
}

// IO allows for simplified access to the http.ResponseWriter and http.Request.
// IO is passed to a Controller's handler function allowing the handler to interact with the http.ResponseWriter and http.Request.
// At the same time, IO allows the handler to interact with frequently used functionality such as rendering templates.
// IO aims to simplify common tasks in the HARMONY web server. Therefore, it is strongly opinionated.
type IO interface {
	// Response returns the http.ResponseWriter for the controller's IO.
	Response() http.ResponseWriter
	// Request returns the http.Request of the request to the controller.
	Request() *http.Request
	// Context returns the context.Context of the request. It is the same context as in the http.Request.
	Context() context.Context
	// RenderTemplate renders a template with the passed in data and writes it to the http.ResponseWriter.
	RenderTemplate(*template.Template, any) error
	// Render renders a template with the passed in data and writes it to the http.ResponseWriter.
	// Render has some convenience over RenderTemplate as it fetches the Templater from the TemplaterStore
	// and then retrieves the specific template from the Templater by name and path.
	// Multiple paths can be provided, and they will be joined together therefore allowing for reusing templates.
	// Example:
	//  	io.Render(formData, "edit.page", "edit-page.go.html", "edit-form.go.html").
	//
	// If the request is an HTMX request, Render renders the template from the partial Templater (PartialTemplateName).
	// Otherwise, it will render the template from the base Templater (BaseTemplateName).
	Render(data any, name string, paths ...string) error
	// Error renders an error page with the first passed in error as the user facing error message.
	// All errors will be logged. At least one error should always be provided as this will be the user facing error message.
	// Error handles HTMX requests by rendering the error template from the partial template.
	//
	// Also adding more errors to improve the meaning of the log entry is highly recommended.
	// If no errors are provided a generic error message is rendered and the error is logged.
	Error(...error) error
	// InlineError is similar to Error, but it renders the error template from the empty template.
	// This allows for rendering the error inline in the page e.g. upon form submission.
	InlineError(...error) error
	// Redirect will send a redirect to the client with the specified status code.
	Redirect(string, int) error
	// IsHTMX returns true if the request is an HTMX request.
	IsHTMX() bool
}

// NewContext creates a new web context using the passed in router, config and templater store.
// The Navigation and TemplateDataExtensions are initialized with NewNavigation and NewExtensions respectively.
func NewContext(router Router, cfg *Cfg, ts TemplaterStore) *Ctx {
	return &Ctx{
		Router:         router,
		Config:         cfg,
		TemplaterStore: ts,
		Navigation:     NewNavigation(),
		Extensions:     NewExtensions(),
	}
}

// NewController creates a new Controller using the passed in hctx.AppCtx, web.Ctx and handler function.
// The handler function is executed when the Controller is used as a handler for serving a request.
// The handler function receives a web.IO which allows for simplified access to the http.ResponseWriter and http.Request.
//
// NewController panics if any of the passed in contexts or the handler function is nil.
func NewController(app *hctx.AppCtx, ctx *Ctx, handler func(io IO) error) http.Handler {
	if app == nil || ctx == nil || handler == nil {
		panic("nil contexts or handler")
	}

	return &Controller{
		appCtx:  app,
		webCtx:  ctx,
		handler: handler,
	}
}

// ServeHTTP implements the http.Handler interface. First, it constructs a web.IO from the http.ResponseWriter and http.Request.
// This web.IO is then used to construct the BaseTemplateData which is set on the web.HIO.
// Then, the handler function is executed with the web.IO.
// If an error occurs, the error is logged and an internal server error is returned to the client.
func (c *Controller) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	io := &HIO{
		writer:  w,
		request: r,
		appCtx:  c.appCtx,
		webCtx:  c.webCtx,
	}

	baseData, err := NewBaseTemplateData(c.appCtx, c.webCtx, io, nil)
	if err != nil {
		c.appCtx.Error(Pkg, "failed to create base template data", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	io.baseData = baseData

	err = c.handler(io)
	if err != nil {
		c.appCtx.Error(Pkg, "internal server error executing handler", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

// Response returns the http.ResponseWriter for the controller's IO.
func (io *HIO) Response() http.ResponseWriter {
	return io.writer
}

// Request returns the http.Request of the request to the controller.
func (io *HIO) Request() *http.Request {
	return io.request
}

// Context returns the context of the request. It is the same context as the one in the http.Request.
func (io *HIO) Context() context.Context {
	return io.request.Context()
}

// Render implements the web.IO interface on HIO by rendering a template by the given name and path with the passed in data.
// For more information on the behaviour of Render see the web.IO interface.
// Render chooses the base Templater based on the request (HTMX or not) and uses RenderTemplate to render the template.
// For an HTMX request, Render uses the partial Templater (PartialTemplateName).
func (io *HIO) Render(data any, name string, paths ...string) error {
	templater, err := io.getBaseTemplater()
	if err != nil {
		return err
	}

	t, err := templater.JoinedTemplate(name, paths...)
	if err != nil {
		return err
	}

	return io.RenderTemplate(t, data)
}

// RenderTemplate implements the web.IO interface on HIO by rendering a template with the passed in data.
// RenderTemplate then writes the executed template to the http.ResponseWriter.
// For more information on the behaviour of RenderTemplate see the web.IO interface.
//
// HIO.RenderTemplate makes the template translatable by calling makeTemplateTranslatable upfront.
// This will add a translation function to the template's function map with a reference to the trans.Translator in the context.
// If makeTemplateTranslatable returns an error, it is logged and the rendering continues. An error is not returned and will not lead to a failed request.
// That is because the template should always be provided with a translation function that just returns the passed in string as-is (fallback).
func (io *HIO) RenderTemplate(t *template.Template, data any) error {
	if err := makeTemplateTranslatable(io.request.Context(), t); err != nil {
		io.appCtx.Warn(Pkg, "failed to make template translatable, likely context does not contain translator", "error", err)
	}

	io.baseData.Data = data

	return util.Wrap(t.Execute(io.writer, io.baseData), "failed to render template")
}

// Error implements the web.IO interface on HIO by rendering an error page with the first passed in error as the user facing error message.
// For more information on the behaviour of Error see the web.IO interface.
// The log entry of all errors is enriched with the request's url, method and header.
//
// Error determines if the request is an HTMX request.
// If it is, it will try to render the error template from the partial template to allow for partial page loads in HTMX.
//
// Then, Error makes the template translatable by calling makeTemplateTranslatable and adding
// the translator from the context to the template's function map (see RenderTemplate).
// If the template is found, it is executed with the first error being the user facing error message.
func (io *HIO) Error(errs ...error) error {
	templater, err := io.getBaseTemplater()
	if err != nil {
		return err
	}

	return io.errs(templater, errs...)
}

// InlineError implements the web.IO interface on HIO by rendering an error inline in the page with the first passed in error as the user facing error message.
// For more information on the behaviour of InlineError see the web.IO interface.
// This method is similar to Error, but it renders the error template from the empty template.
func (io *HIO) InlineError(errs ...error) error {
	templater, err := io.getBaseInlineTemplater()
	if err != nil {
		return err
	}

	return io.errs(templater, errs...)
}

// Redirect will send a redirect to the client.
func (io *HIO) Redirect(url string, code int) error {
	http.Redirect(io.writer, io.request, url, code)
	return nil
}

// IsHTMX returns true if the request is an HTMX request.
func (io *HIO) IsHTMX() bool {
	return io.baseData.HTMX
}

// errs is a helper function for Error and InlineError.
// It renders the error template from the passed in templater with the first passed in error as the user facing error message.
// It also adds the request's url, method and header to the log entry of all errors.
// errs also makes the template translatable by calling makeTemplateTranslatable.
func (io *HIO) errs(templater Templater, errs ...error) error {
	if len(errs) == 0 {
		errs = append(errs, fmt.Errorf("harmony.error.generic-reload"))
	}

	for _, err := range errs {
		io.appCtx.Error(Pkg, "error in controller", err, "url", io.request.URL.String(), "method", io.request.Method)
	}

	e := errs[0]

	errTemplate, err := templater.Template("error", "error.go.html")
	if err != nil {
		return err
	}

	if err = makeTemplateTranslatable(io.request.Context(), errTemplate); err != nil {
		io.appCtx.Warn(Pkg, "failed to make template translatable, likely context does not contain translator", "error", err)
	}

	io.baseData.Data = e.Error()

	return errTemplate.Execute(io.writer, io.baseData)
}

// getBaseTemplater returns the base Templater based on the request (HTMX or not).
// If the request is an HTMX request, it returns the partial Templater (PartialTemplateName).
// Otherwise, it will return the base Templater (BaseTemplateName).
func (io *HIO) getBaseTemplater() (Templater, error) {
	if io.IsHTMX() {
		return io.webCtx.TemplaterStore.Templater(PartialTemplateName)
	}

	return io.webCtx.TemplaterStore.Templater(BaseTemplateName)
}

// getBaseInlineTemplater returns the base Templater for inline errors which is the empty Templater (EmptyTemplateName).
func (io *HIO) getBaseInlineTemplater() (Templater, error) {
	return io.webCtx.TemplaterStore.Templater(EmptyTemplateName)
}

// MountFileServer registers a file server with a config on a router.
func MountFileServer(r Router, cfg *FileServerCfg) {
	route := cfg.Route

	// Path Validation
	if strings.ContainsAny(route, "{}*") {
		panic("FileServer does not permit any URL parameters.")
	}

	// Path Adjustment and Redirection
	if route != "/" && route[len(route)-1] != '/' {
		r.Get(route, http.RedirectHandler(route+"/", 301).ServeHTTP)
		route += "/"
	}

	// Adjust the route to include a wildcard
	routeWithWildcard := route + "*"

	// Handling of GET requests
	r.Get(routeWithWildcard, func(w http.ResponseWriter, r *http.Request) {
		pathPrefix := strings.TrimSuffix(route, "/*")
		fs := http.StripPrefix(pathPrefix, http.FileServer(http.Dir(cfg.Root)))
		fs.ServeHTTP(w, r)
	})
}

// Serve starts a web server on a router using the address and port specified in the config.
func Serve(r Router, cfg *ServerCfg) error {
	return http.ListenAndServe(fmt.Sprintf("%s:%s", cfg.Addr, cfg.Port), r)
}

// ReadForm reads the form values from a request and populates the fields of a struct pointed to by 'data'.
// It expects 'data' to be a pointer to a struct, otherwise it panics. It only populates exported fields.
// If a validator is provided, it will be used to validate the struct after the values have been populated.
//
// It returns an ErrInternalReadForm error if the request could not be parsed.
// If the struct is invalid the first returned error will be nil and the returned slice of errors will contain the validation errors.
//
// ReadForm will panic if 'data' is not a pointer to a struct.
//
// ReadForm first parses the form values from the request and then populates the struct using ValuesIntoStruct function.
// ValuesIntoStruct uses reflection and does not yet support nested structs. For more information see ValuesIntoStruct.
func ReadForm(r *http.Request, data any, validator validation.V) (error, []error) {
	if !isPointerToStruct(data) {
		panic(ErrNotPointerToStruct)
	}

	err := r.ParseForm()
	if err != nil {
		return errors.Join(ErrInternalReadForm, err), nil
	}

	values := r.Form
	if err := ValuesIntoStruct(values, data); err != nil {
		return errors.Join(ErrInternalReadForm, err), nil
	}

	if validator == nil {
		return nil, nil
	}

	hardErr, validationErrs := validator.ValidateStruct(data)
	if hardErr != nil {
		return errors.Join(ErrInternalReadForm, hardErr), nil
	}

	return nil, validationErrs
}

// ValuesIntoStruct populates the fields of a struct pointed to by 'data' with corresponding values from 'values'.
// It expects 'data' to be a pointer to a struct, and it only populates exported fields.
// If multiple values are provided for single value items (e.g. int, string, bool), only the first one is used.
// If a field is not present in 'values' or the value can not be set or converted to the corresponding datatype it is skipped.
// ValuesIntoStruct returns a ErrNotPointerToStruct error if 'data' is not a pointer to a struct.
//
// ValuesIntoStruct should primarily be used through the ReadForm function.
//
// TODO add support for nested structs
// TODO add support for other types (e.g. slices, maps)
// TODO allow for custom field names via struct tags
func ValuesIntoStruct(values url.Values, data any) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%writer: %v", ErrUnexpectedReflection, r)
		}
	}()

	if !isPointerToStruct(data) {
		return ErrNotPointerToStruct
	}

	dataType := reflect.TypeOf(data).Elem()
	dataValue := reflect.ValueOf(data).Elem()

	for i := 0; i < dataType.NumField(); i++ {
		fieldType := dataType.Field(i)
		fieldValue := dataValue.Field(i)

		if !fieldValue.CanSet() {
			continue
		}

		if val, ok := values[fieldType.Name]; ok {
			setValues(fieldValue, val)
		}
	}

	return nil
}

// isPointerToStruct checks if the input is a pointer to a struct.
func isPointerToStruct(input any) bool {
	t := reflect.TypeOf(input)
	return t.Kind() == reflect.Ptr && t.Elem().Kind() == reflect.Struct
}

// setValues sets the values of a field based on the field's type.
// The function uses reflection and accounts for pointers.
// If multiple values are provided for single value items (e.g. int, string, bool), only the first one is used.
func setValues(field reflect.Value, val []string) {
	kind := field.Kind()

	if kind == reflect.Ptr {
		if field.IsNil() {
			field.Set(reflect.New(field.Type().Elem()))
		}
		field = field.Elem()
	}

	// handle slices, maps, arrays, etc. here

	if len(val) < 1 {
		return
	}
	singleVal := val[0]

	switch kind {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		setIntValue(field, singleVal)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		setUintValue(field, singleVal)
	case reflect.Float32, reflect.Float64:
		setFloatValue(field, singleVal)
	case reflect.Bool:
		setBoolValue(field, singleVal)
	case reflect.String:
		setStringValue(field, singleVal)
	}
}

// setIntValue tries to set an int on a reflect.Value.
// It might panic if the reflect.Value is not settable or not an int.
func setIntValue(field reflect.Value, val string) {
	if intVal, err := strconv.Atoi(val); err == nil {
		field.SetInt(int64(intVal))
	}
}

// setUintValue tries to set an uint on a reflect.Value.
// It might panic if the reflect.Value is not settable or not an uint.
func setUintValue(field reflect.Value, val string) {
	if uintVal, err := strconv.ParseUint(val, 10, 64); err == nil {
		field.SetUint(uintVal)
	}
}

// setFloatValue tries to set a float on a reflect.Value.
// It might panic if the reflect.Value is not settable or not a float.
func setFloatValue(field reflect.Value, val string) {
	if floatVal, err := strconv.ParseFloat(val, 64); err == nil {
		field.SetFloat(floatVal)
	}
}

// setBoolValue tries to set a boolean on a reflect.Value.
// It might panic if the reflect.Value is not settable or not a boolean.
func setBoolValue(field reflect.Value, val string) {
	if boolVal, err := strconv.ParseBool(val); err == nil {
		field.SetBool(boolVal)
	}
}

// setStringValue sets a string on a reflect.Value.
// It might panic if the reflect.Value is not settable or not a string.
func setStringValue(field reflect.Value, val string) {
	field.SetString(val)
}
