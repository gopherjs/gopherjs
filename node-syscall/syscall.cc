#include <memory>
#include <stdexcept>
#include <vector>
#include <cstdlib>
#include <node.h>
#include <v8.h>
#include <unistd.h>
#include <sys/syscall.h>
#include <errno.h>
#include <iostream>

using namespace v8;

// arena stores buffers we allocate for data passed to syscalls.
//
// This object lives for the duration of Syscall() or Syscall6() and correctly
// frees all allocated buffers at the end. This is necessary to avoid memory
// leaks on each call.
class arena {
  std::vector<std::vector<uint8_t>> allocs_;
public:
  arena() = default;
  virtual ~arena() = default;
  arena(const arena& a) = delete;

  void* allocate(size_t n) {
    allocs_.emplace_back(n); // Allocate a new vector of n byte size.
    return allocs_[allocs_.size() - 1].data(); // Return the pointer to its data buffer;
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

// Transforms a JS value into a native value that can be passed to the syscall() call.
intptr_t toNative(Local<Context> ctx, arena& a, Local<Value> value) {
  if (value.IsEmpty()) {
    return 0;
  }
  if (value->IsArrayBufferView()) {
    Local<ArrayBufferView> view = Local<ArrayBufferView>::Cast(value);
    void* native = a.allocate(view->ByteLength());
    view->CopyContents(native, view->ByteLength());
    return reinterpret_cast<intptr_t>(native);
  }
  if (value->IsArray()) {
    Local<Array> array = Local<Array>::Cast(value);
    intptr_t* native = reinterpret_cast<intptr_t*>(a.allocate(array->Length() * sizeof(intptr_t)));
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

extern "C" NODE_MODULE_EXPORT void
NODE_MODULE_INITIALIZER(Local<Object> exports,
                        Local<Value> module,
                        Local<Context> context) {
  NODE_SET_METHOD(exports, "Syscall", Syscall);
  NODE_SET_METHOD(exports, "Syscall6", Syscall6);
}
