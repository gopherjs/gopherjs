#include <memory>
#include <stdexcept>
#include <vector>
#include <cstdlib>
#include <node.h>
#include <v8.h>
#include <unistd.h>
#include <sys/syscall.h>
#include <errno.h>

using namespace v8;

#if NODE_MAJOR_VERSION == 0
#define ARRAY_BUFFER_DATA_OFFSET 23
#else
#define ARRAY_BUFFER_DATA_OFFSET 31
#endif

// arena stores buffers we allocate for data passed to syscalls.
//
// This object lives for the duration of Syscall() or Syscall6() and correctly
// frees all allocated buffers at the end. This is necessary to avoid memory
// leaks on each call.
class arena {
  std::vector<std::unique_ptr<intptr_t[]>> allocs_;
public:
  arena() = default;
  virtual ~arena() = default;
  arena(const arena& a) = delete;

  intptr_t* allocate(size_t n) {
    allocs_.emplace_back(new intptr_t[n]);
    return allocs_.end()->get();
  }
};

// Cast value to integer or throw an exception.
//
// The exception must be explicitly caught up the call stack, since the Node API
// this extension is using doesn't seem to handle exceptions.
Local<Integer> integerOrDie(Local<Context> ctx, Local<Value> value) {
  Local<Integer> integer;
  if (value->ToInteger(ctx).ToLocal(&integer)) {
    return integer;
  }
  throw std::runtime_error("expected integer, got something else");
}

intptr_t toNative(Local<Context> ctx, arena& a, Local<Value> value) {
  if (value.IsEmpty()) {
    return 0;
  }
  if (value->IsArrayBufferView()) {
    Local<ArrayBufferView> view = Local<ArrayBufferView>::Cast(value);
    return *reinterpret_cast<intptr_t*>(*reinterpret_cast<char**>(*view->Buffer()) + ARRAY_BUFFER_DATA_OFFSET) + view->ByteOffset(); // ugly hack, because of https://codereview.chromium.org/25221002
  }
  if (value->IsArray()) {
    Local<Array> array = Local<Array>::Cast(value);
    intptr_t* native = a.allocate(array->Length());
    for (uint32_t i = 0; i < array->Length(); i++) {
      native[i] = toNative(ctx, a, array->Get(ctx, i).ToLocalChecked());
    }
    return reinterpret_cast<intptr_t>(native);
  }

  return static_cast<intptr_t>(static_cast<int32_t>(integerOrDie(ctx, value)->Value()));
}

void Syscall(const FunctionCallbackInfo<Value>& info) {
  arena a;
  Isolate* isolate = info.GetIsolate();
  Local<Context> ctx = isolate->GetCurrentContext();

  try {
    int trap = integerOrDie(ctx, info[0])->Value();
    int r1 = 0, r2 = 0;
    switch (trap) {
    case SYS_fork:
      r1 = fork();
      break;
    case SYS_pipe:
      int fd[2];
      r1 = pipe(fd);
      if (r1 == 0) {
        r1 = fd[0];
        r2 = fd[1];
      }
      break;
    default:
      r1 = syscall(
        trap,
        toNative(ctx, a, info[1]),
        toNative(ctx, a, info[2]),
        toNative(ctx, a, info[3])
      );
      break;
    }
    int err = 0;
    if (r1 < 0) {
      err = errno;
    }
    Local<Array> res = Array::New(isolate, 3);
    res->Set(ctx, 0, Integer::New(isolate, r1)).ToChecked();
    res->Set(ctx, 1, Integer::New(isolate, r2)).ToChecked();
    res->Set(ctx, 2, Integer::New(isolate, err)).ToChecked();
    info.GetReturnValue().Set(res);
  } catch (std::exception& e) {
    auto message = String::NewFromUtf8(isolate, e.what(), NewStringType::kNormal).ToLocalChecked();
    isolate->ThrowException(Exception::TypeError(message));
    return;
  }
}

void Syscall6(const FunctionCallbackInfo<Value>& info) {
  arena a;
  Isolate* isolate = info.GetIsolate();
  Local<Context> ctx = Context::New(isolate);

  try {
    int r = syscall(
      integerOrDie(ctx, info[0])->Value(),
      toNative(ctx, a, info[1]),
      toNative(ctx, a, info[2]),
      toNative(ctx, a, info[3]),
      toNative(ctx, a, info[4]),
      toNative(ctx, a, info[5]),
      toNative(ctx, a, info[6])
    );
    int err = 0;
    if (r < 0) {
      err = errno;
    }
    Local<Array> res = Array::New(isolate, 3);
    res->Set(ctx, 0, Integer::New(isolate, r)).ToChecked();
    res->Set(ctx, 1, Integer::New(isolate, 0)).ToChecked();
    res->Set(ctx, 2, Integer::New(isolate, err)).ToChecked();
    info.GetReturnValue().Set(res);
  } catch (std::exception& e) {
    auto message = String::NewFromUtf8(isolate, e.what(), NewStringType::kNormal).ToLocalChecked();
    isolate->ThrowException(Exception::TypeError(message));
    return;
  }
}

void init(Local<Object> exports) {
  NODE_SET_METHOD(exports, "Syscall", Syscall);
  NODE_SET_METHOD(exports, "Syscall6", Syscall6);
}

NODE_MODULE(syscall, init);
