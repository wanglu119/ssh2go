package libssh

/*
#cgo pkg-config: libssh
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <inttypes.h>
#include <sys/types.h>
#include <libssh/libssh.h>
#include <libssh/server.h>
#include <libssh/callbacks.h>
#include "callbacks.h"

extern void set_session_message_callback(ssh_session session, void *userdata);
*/
import "C"
import (
	"fmt"
	"unsafe"
)

const (
	SSH_BIND_OPTIONS_BINDADDR          = C.SSH_BIND_OPTIONS_BINDADDR
	SSH_BIND_OPTIONS_BINDPORT          = C.SSH_BIND_OPTIONS_BINDPORT
	SSH_BIND_OPTIONS_BINDPORT_STR      = C.SSH_BIND_OPTIONS_BINDPORT_STR
	SSH_BIND_OPTIONS_HOSTKEY           = C.SSH_BIND_OPTIONS_HOSTKEY
	SSH_BIND_OPTIONS_DSAKEY            = C.SSH_BIND_OPTIONS_DSAKEY
	SSH_BIND_OPTIONS_RSAKEY            = C.SSH_BIND_OPTIONS_RSAKEY
	SSH_BIND_OPTIONS_BANNER            = C.SSH_BIND_OPTIONS_BANNER
	SSH_BIND_OPTIONS_LOG_VERBOSITY     = C.SSH_BIND_OPTIONS_LOG_VERBOSITY
	SSH_BIND_OPTIONS_LOG_VERBOSITY_STR = C.SSH_BIND_OPTIONS_LOG_VERBOSITY_STR
	SSH_BIND_OPTIONS_ECDSAKEY          = C.SSH_BIND_OPTIONS_ECDSAKEY
)

type Bind struct {
	ptr C.ssh_bind
}

type BindIncomingConnectionCallback interface {
	OnBindIncomingConnection(bind Bind)
}

func wrapBindIncomingConnectionCallback(cb BindIncomingConnectionCallback) C.ssh_bind_incoming_connection_callback {
	ptr := NULL
	if cb != nil {
		wrapper := func(bind C.ssh_bind, userdata unsafe.Pointer) {
			cb.OnBindIncomingConnection(Bind{bind})
		}
		ptr = unsafe.Pointer(&wrapper)
	}
	return C.ssh_bind_incoming_connection_callback(ptr)
}

func NewBind() (Bind, error) {
	bind := C.ssh_bind_new()
	if bind == nil {
		return Bind{}, apiError("ssh_bind_new", "NULL")
	}
	return Bind{bind}, nil
}

func (b Bind) Free() {
	C.ssh_bind_free(b.ptr)
}

func (b Bind) SetOption(optionType int, value interface{}) error {
	var ptr unsafe.Pointer
	switch optionType {
	case SSH_BIND_OPTIONS_BINDADDR,
		SSH_BIND_OPTIONS_BINDPORT,
		SSH_BIND_OPTIONS_LOG_VERBOSITY:
		val, ok := value.(int)
		if ok {
			ptr = unsafe.Pointer(&val)
		}

	case SSH_BIND_OPTIONS_BINDPORT_STR,
		SSH_BIND_OPTIONS_HOSTKEY,
		SSH_BIND_OPTIONS_DSAKEY,
		SSH_BIND_OPTIONS_RSAKEY,
		SSH_BIND_OPTIONS_BANNER,
		SSH_BIND_OPTIONS_LOG_VERBOSITY_STR,
		SSH_BIND_OPTIONS_ECDSAKEY:
		val, ok := value.(string)
		if ok {
			v := CString(val)
			defer v.Free()
			ptr = unsafe.Pointer(v.Ptr)
		}
	}
	if ptr == nil {
		return fmt.Errorf("Illegal parameter type for ssh_bind_options_set()")
	}
	return apiError("ssh_bind_options_set",
		C.ssh_bind_options_set(b.ptr, C.enum_ssh_bind_options_e(optionType), ptr))
}

func (b Bind) Listen() error {
	return apiError("ssh_bind_listen", C.ssh_bind_listen(b.ptr))
}

type BindCallbacks struct {
	cbsptr               C.ssh_bind_callbacks_wrapper
	OnIncomingConnection func(bind Bind)
}

func (callbacks *BindCallbacks) Free() {
	if callbacks.cbsptr != nil {
		C.free(unsafe.Pointer(callbacks.cbsptr))
		callbacks.cbsptr = nil
	}
}

//export bind_incoming_connection_callback
func bind_incoming_connection_callback(bind C.ssh_bind, userdata unsafe.Pointer) {
	callbacks := (*BindCallbacks)(userdata)
	callbacks.OnIncomingConnection(Bind{bind})
}

func (b Bind) SetCallback(callbacks *BindCallbacks) error {
	callbacks.cbsptr = C.new_bind_callbacks()
	callbacks.cbsptr.userdata = unsafe.Pointer(callbacks)
	if callbacks.OnIncomingConnection != nil {
		C.install_bind_incoming_connection_callback(&callbacks.cbsptr.callbacks)
	}
	return apiError("ssh_bind_set_callbacks", C.set_bind_callbacks(b.ptr, callbacks.cbsptr))
}

func (b Bind) SetBlocking(blocking bool) {
	C.ssh_bind_set_blocking(b.ptr, CBool(blocking))
}

func (b Bind) GetFd() (int, error) {
	fd := C.ssh_bind_get_fd(b.ptr)
	return int(fd), apiError("ssh_bind_get_fd", fd)
}

func (b Bind) SetFd(socketFd int) {
	C.ssh_bind_set_fd(b.ptr, C.socket_t(socketFd))
}

func (b Bind) SetFdToAccept() {
	C.ssh_bind_fd_toaccept(b.ptr)
}

// Accept an incoming ssh connection and initialize the session.
//
// session:
//  A preallocated ssh session
func (b Bind) Accept(session Session) error {
	return apiError("ssh_bind_accept", C.ssh_bind_accept(b.ptr, session.ptr))
}

func (b Bind) AcceptFd(session Session, socketFd int) error {
	return apiError("ssh_bind_accept_fd", C.ssh_bind_accept(b.ptr, session.ptr))
}

func (s Session) HandleKeyExchange() error {
	return apiError("ssh_handle_key_exchange", C.ssh_handle_key_exchange(s.ptr))
}

// methods:
//  SSH_AUTH_METHOD_UNKNOWN 0
//  SSH_AUTH_METHOD_NONE 0x0001
//  SSH_AUTH_METHOD_PASSWORD 0x0002
//  SSH_AUTH_METHOD_PUBLICKEY 0x0004
//  SSH_AUTH_METHOD_HOSTBASED 0x0008
//  SSH_AUTH_METHOD_INTERACTIVE 0x0010
//  SSH_AUTH_METHOD_GSSAPI_MIC 0x0020
func (s Session) SetAuthMethods(methods int) {
	C.ssh_set_auth_methods(s.ptr, C.int(methods))
}

// for server

type BindMessageCallback struct {
	OnBindMessage func(session Session, msg Message) int
}

//export bind_message_callback
func bind_message_callback(sess C.ssh_session, msg C.ssh_message, userdata unsafe.Pointer) C.int {
	callback := (*BindMessageCallback)(userdata)
	return C.int(callback.OnBindMessage(Session{sess}, Message{msg}))
}

func (s Session) SetMessageCallback(callback *BindMessageCallback) {
	C.set_session_message_callback(s.ptr, unsafe.Pointer(callback))
}

func (s Session) ExecuteMessageCallbacks() error {
	return apiError("ssh_execute_message_callbacks", C.ssh_execute_message_callbacks(s.ptr))
}
