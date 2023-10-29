// Package web provides a web server implementation for HARMONY and declaring its interface.
// Also, the web package handles utility and required web functionality to allow domain packages
// to easily extend upon this package and allow web communication.
package web

import (
	"context"
	"errors"
	"fmt"
	"github.com/org-harmony/harmony/src/core/hctx"
	"github.com/org-harmony/harmony/src/core/trace"
	"github.com/org-harmony/harmony/src/core/util"
	"github.com/org-harmony/harmony/src/core/validation"
	"html/template"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
)

const Pkg = "sys.web"

var (
	// ErrNotPointerToStruct is returned when the input is not a pointer to a struct.
	ErrNotPointerToStruct = errors.New("input is not a pointer to a struct")
	// ErrUnexpectedReflection is returned when an unexpected reflection error occurs.
	ErrUnexpectedReflection = errors.New("unexpected reflection error")
	// ErrInvalidStruct is returned when the input struct is invalid with validation errors.
	ErrInvalidStruct = errors.New("invalid struct")
	// ErrInternalReadForm is returned when an internal error occurs while reading the form.
	ErrInternalReadForm = errors.New("internal error reading form")
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

// Ctx is the web context.
// It contains the router, the config and the templater store.
type Ctx struct {
	Router         Router
	Config         *Cfg
	TemplaterStore TemplaterStore
}

// Controller is convenience struct for handling web requests.
// The Controller is aware of the application context and the web context.
// The Controller implements the http.Handler interface and can therefore be used as a handler.
type Controller struct {
	app     *hctx.AppCtx
	ctx     *Ctx
	handler func(io IO) error
}

// HIO is a web.IO allowing for simplified access to the http.ResponseWriter and http.Request.
type HIO struct {
	w  http.ResponseWriter
	r  *http.Request
	l  trace.Logger
	t  TemplaterStore
	rt Router
}

// IO allows for simplified access to the http.ResponseWriter and http.Request.
// IO is passed to a Controller's handler function allowing the handler to interact with the http.ResponseWriter and http.Request.
// At the same time, IO allows the handler to interact with frequently used functionality such as logging and rendering.
type IO interface {
	// Response returns the http.ResponseWriter for the controller's IO.
	Response() http.ResponseWriter
	// Request returns the http.Request of the request to the controller.
	Request() *http.Request
	// Context returns the context.Context of the request. It is the same context.Context as the one in the http.Request.
	Context() context.Context
	// Logger returns the trace.Logger used by the application.
	Logger() trace.Logger
	// TemplaterStore returns the TemplaterStore used by the application containing the Templaters.
	TemplaterStore() TemplaterStore
	// Router returns the router used by the application.
	Router() Router
	// Render renders a template with the passed in data and writes it to the http.ResponseWriter.
	Render(*template.Template, any) error
	// Error renders an error page with the first passed in error as the user facing error message.
	// All errors will be logged. If no errors are provided a generic error message is rendered and the error is logged.
	// TODO add support for htmx errors => only rendering the error template from empty templater
	Error(...error) error
	// Redirect will send a redirect to the client with the specified status code.
	Redirect(string, int) error
	// Template returns a template by a name from the TemplateStore.
	// Template just wraps the call to TemplaterStore and consecutive Template call.
	Template(store, template, path string) (*template.Template, error)
}

// NewContext creates a new web context using the passed in router, config and templater store.
func NewContext(router Router, cfg *Cfg, ts TemplaterStore) *Ctx {
	return &Ctx{
		Router:         router,
		Config:         cfg,
		TemplaterStore: ts,
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
		app:     app,
		ctx:     ctx,
		handler: handler,
	}
}

// ServeHTTP implements the http.Handler interface. It executes the handler function and handles any errors.
// If an error occurs, the error is logged and an internal server error is returned to the client.
func (c *Controller) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	io := &HIO{
		w:  w,
		r:  r,
		l:  c.app,
		t:  c.ctx.TemplaterStore,
		rt: c.ctx.Router,
	}

	err := c.handler(io)
	if err != nil {
		c.app.Error(Pkg, "internal server error executing handler", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

// Response returns the http.ResponseWriter for the controller's IO.
func (h *HIO) Response() http.ResponseWriter {
	return h.w
}

// Request returns the http.Request of the request to the controller.
func (h *HIO) Request() *http.Request {
	return h.r
}

// Context returns the context.Context of the request. It is the same context.Context as the one in the http.Request.
func (h *HIO) Context() context.Context {
	return h.r.Context()
}

// Logger returns the trace.Logger used by the application.
func (h *HIO) Logger() trace.Logger {
	return h.l
}

// TemplaterStore returns the TemplaterStore used by the application containing the Templaters.
func (h *HIO) TemplaterStore() TemplaterStore {
	return h.t
}

// Router returns the router used by the application.
func (h *HIO) Router() Router {
	return h.rt
}

// Render renders a template with the passed in data and writes it to the http.ResponseWriter.
// Upfront, Render makes the template translatable by calling makeTemplateTranslatable.
// This will add a translation function to the template's function map with a reference to the trans.Translator in the context.
// If makeTemplateTranslatable returns an error, it is logged and the rendering continues. An error is not returned and will not lead to a failed request.
// That is because the template should always be provided with a translation function that just returns the passed in string as-is (fallback).
func (h *HIO) Render(t *template.Template, data any) error {
	if err := makeTemplateTranslatable(h.r.Context(), t); err != nil {
		h.l.Warn(Pkg, "failed to make template translatable, likely context does not contain translator", "error", err)
	}

	return util.Wrap(t.Execute(h.w, data), "failed to render template")
}

// Error renders an error page with the first passed in error as the user facing error message.
// All errors will be logged. If no errors are provided a generic error message is rendered and the error is logged.
// The log entry is enriched with the request's url, method and header.
//
// Error first tries to get the error template from the template store. If that fails, it returns an error.
// Then, it makes the template translatable by calling makeTemplateTranslatable and adding
// the translator from the context to the template's function map (see Render).
// If the template is found, it is executed with the first error being the user facing error message.
func (h *HIO) Error(errs ...error) error {
	if len(errs) == 0 {
		errs = append(errs, fmt.Errorf("harmony.error.generic"))
	}

	for _, err := range errs {
		h.l.Error(Pkg, "error in controller", err, "url", h.r.URL.String(), "method", h.r.Method, "header", h.r.Header)
	}

	e := errs[0]

	errTemplater, err := h.t.Templater(ErrorTemplateName)
	if err != nil {
		return err
	}

	errTemplate, err := errTemplater.Base()
	if err != nil {
		return err
	}

	if err = makeTemplateTranslatable(h.r.Context(), errTemplate); err != nil {
		h.l.Warn(Pkg, "failed to make template translatable, likely context does not contain translator", "error", err)
	}

	return errTemplate.Execute(h.w, map[string]string{"Err": e.Error()})
}

// Redirect will send a redirect to the client.
func (h *HIO) Redirect(url string, code int) error {
	http.Redirect(h.w, h.r, url, code)
	return nil
}

// Template returns a template by a name from the TemplateStore.
// Template just wraps the call to TemplaterStore and consecutive Template call.
func (h *HIO) Template(store, template, path string) (*template.Template, error) {
	templaterStore, err := h.t.Templater(store)
	if err != nil {
		return nil, err
	}

	return templaterStore.Template(template, path)
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
// It will return an error if the request could not be parsed or if the struct is invalid.
// For an invalid struct you will receive an ErrInvalidStruct followed by the validation errors.
// For all other errors you will receive an ErrInternalReadForm followed by the error.
//
// ReadForm will panic if 'data' is not a pointer to a struct.
//
// ReadForm first parses the form values from the request and then populates the struct using ValuesIntoStruct function.
// ValuesIntoStruct uses reflection and does not yet support nested structs. For more information see ValuesIntoStruct.
func ReadForm(r *http.Request, data any, validator validation.V) error {
	if !isPointerToStruct(data) {
		panic(ErrNotPointerToStruct)
	}

	err := r.ParseForm()
	if err != nil {
		return errors.Join(ErrInternalReadForm, err)
	}

	values := r.Form
	if err := ValuesIntoStruct(values, data); err != nil {
		return errors.Join(ErrInternalReadForm, err)
	}

	if validator == nil {
		return nil
	}

	hardErr, validationErrs := validator.ValidateStruct(data)
	if hardErr != nil {
		return errors.Join(ErrInternalReadForm, hardErr)
	}

	if len(validationErrs) > 0 {
		return errors.Join(append([]error{ErrInvalidStruct}, validationErrs...)...)
	}

	return nil
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
			err = fmt.Errorf("%w: %v", ErrUnexpectedReflection, r)
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
