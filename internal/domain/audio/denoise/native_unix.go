//go:build cgo && (linux || darwin)

package denoise

/*
#cgo linux LDFLAGS: -ldl
#include <dlfcn.h>
#include <stdint.h>
#include <stdlib.h>

typedef struct {
 void *rnlib, *splib, *rn, *up, *down;
 void *(*create)(void);
 void (*destroy)(void*);
 float (*process)(void*,float*,const float*);
 void *(*res_init)(uint32_t,uint32_t,uint32_t,int,int*);
 void (*res_destroy)(void*);
 int (*res_process)(void*,uint32_t,const float*,uint32_t*,float*,uint32_t*);
 int (*res_latency)(void*);
} noise_stream;

static void noise_close(noise_stream *s) {
 if (!s) return;
 if (s->rn) s->destroy(s->rn);
 if (s->up) s->res_destroy(s->up);
 if (s->down) s->res_destroy(s->down);
 if (s->rnlib) dlclose(s->rnlib);
 if (s->splib) dlclose(s->splib);
 free(s);
}
static noise_stream *noise_open(const char *rnpath, const char *sppath) {
 noise_stream *s = calloc(1,sizeof(*s));
 if (!s) return NULL;
 s->rnlib=dlopen(rnpath,RTLD_NOW|RTLD_LOCAL);
 s->splib=dlopen(sppath,RTLD_NOW|RTLD_LOCAL);
 if (!s->rnlib || !s->splib) goto fail;
 #define RN(field,name) s->field=dlsym(s->rnlib,name); if (!s->field) goto fail
 #define SP(field,name) s->field=dlsym(s->splib,name); if (!s->field) goto fail
 RN(create,"rnnoise_create"); RN(destroy,"rnnoise_destroy"); RN(process,"rnnoise_process_frame");
 SP(res_init,"speex_resampler_init"); SP(res_destroy,"speex_resampler_destroy");
 SP(res_process,"speex_resampler_process_float"); SP(res_latency,"speex_resampler_get_output_latency");
 int err=0;
 s->rn=s->create();
 s->up=s->res_init(1,16000,48000,5,&err);
 if (err) goto fail;
 s->down=s->res_init(1,48000,16000,5,&err);
 if (err || !s->rn || !s->up || !s->down) goto fail;
 // Initialize v0.1's shared FFT tables while Go holds nativeInitMu.
 float silence[480]={0}, discard[480];
 s->process(s->rn,discard,silence);
 s->destroy(s->rn); s->rn=s->create();
 if (!s->rn) goto fail;
 return s;
 fail: noise_close(s); return NULL;
}
static int noise_delay(noise_stream *s) {
 return 160 + s->res_latency(s->up)/3 + s->res_latency(s->down);
}
static int noise_process(noise_stream *s, const float *in, float *out) {
 float up[480], clean[480];
 uint32_t ni=160,no=480;
 int err=s->res_process(s->up,0,in,&ni,up,&no);
 if (err || ni!=160 || no!=480) return -1;
 // RNNoise v0.1 uses float PCM in int16 units, not normalized [-1,1].
 for (int i=0;i<480;i++) up[i]*=32768.f;
 s->process(s->rn,clean,up);
 for (int i=0;i<480;i++) clean[i]/=32768.f;
 ni=480; no=160;
 err=s->res_process(s->down,0,clean,&ni,out,&no);
 return (err || ni!=480 || no!=160) ? -2 : 0;
}
*/
import "C"

import (
	"fmt"
	"os"
	"runtime"
	"sync"
	"unsafe"
)

type nativeStream struct{ ptr *C.noise_stream }

var nativeInitMu sync.Mutex

// Keep each loaded library resident: RNNoise's shared FFT tables live for the
// process lifetime. Per-stream states are still destroyed on Close.
var nativeLibraries = map[string]unsafe.Pointer{}

func newNative() (*nativeStream, error) {
	rn, sp := "libxiaozhi_rnnoise_v01.so", "libspeexdsp.so.1"
	if runtime.GOOS == "darwin" {
		rn, sp = "libxiaozhi_rnnoise_v01.dylib", "libspeexdsp.dylib"
	}
	if path := os.Getenv("XIAOZHI_RNNOISE_LIBRARY"); path != "" {
		rn = path
	}
	if path := os.Getenv("XIAOZHI_SPEEXDSP_LIBRARY"); path != "" {
		sp = path
	}
	a, b := C.CString(rn), C.CString(sp)
	defer C.free(unsafe.Pointer(a))
	defer C.free(unsafe.Pointer(b))
	nativeInitMu.Lock()
	defer nativeInitMu.Unlock()
	for path, name := range map[string]*C.char{rn: a, sp: b} {
		if nativeLibraries[path] == nil {
			handle := C.dlopen(name, C.RTLD_NOW|C.RTLD_LOCAL)
			if handle == nil {
				return nil, fmt.Errorf("cannot load audio library %s", path)
			}
			nativeLibraries[path] = handle
		}
	}
	p := C.noise_open(a, b)
	if p == nil {
		return nil, fmt.Errorf("cannot initialize pinned RNNoise v0.1/SpeexDSP (%s, %s)", rn, sp)
	}
	return &nativeStream{ptr: p}, nil
}
func (n *nativeStream) process(pcm []float32) ([]float32, error) {
	out := make([]float32, 160)
	input := make([]float32, 160)
	for i, x := range pcm {
		if x > 1 {
			x = 1
		}
		if x < -1 {
			x = -1
		}
		input[i] = x
	}
	if C.noise_process(n.ptr, (*C.float)(unsafe.Pointer(&input[0])), (*C.float)(unsafe.Pointer(&out[0]))) != 0 {
		return nil, fmt.Errorf("RNNoise resampler frame mismatch")
	}
	for i, x := range out {
		if x > 1 {
			out[i] = 1
		}
		if x < -1 {
			out[i] = -1
		}
	}
	return out, nil
}
func (n *nativeStream) close() {
	nativeInitMu.Lock()
	defer nativeInitMu.Unlock()
	C.noise_close(n.ptr)
	n.ptr = nil
}

func (n *nativeStream) latencySamples() int { return int(C.noise_delay(n.ptr)) }
