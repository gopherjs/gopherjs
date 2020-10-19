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

intptr_t toNative(Local<Value> value, v8::Isolate* isolate) {
  if (value.IsEmpty()) {
    return 0;
  }
  if (value->IsArrayBufferView()) {
    Local<ArrayBufferView> view = Local<ArrayBufferView>::Cast(value);
    return *reinterpret_cast<intptr_t*>(*reinterpret_cast<char**>(*view->Buffer()) + ARRAY_BUFFER_DATA_OFFSET) + view->ByteOffset(); // ugly hack, because of https://codereview.chromium.org/25221002
  }
  if (value->IsArray()) {
    Local<Array> array = Local<Array>::Cast(value);
    intptr_t* native = reinterpret_cast<intptr_t*>(malloc(array->Length() * sizeof(intptr_t))); // TODO memory leak
    for (uint32_t i = 0; i < array->Length(); i++) {
      native[i] = toNative(array->Get(i),isolate);
    }
    return reinterpret_cast<intptr_t>(native);
  }
  Local<Context> context = Context::New(isolate);
  Local<Number> number;
  if (!value->ToInteger(context).ToLocal(&number)) {
    return 0;
  }
  return static_cast<intptr_t>(static_cast<int32_t>(number->Value()));
}

void Syscall(const FunctionCallbackInfo<Value>& info) {
  Isolate* isolate = info.GetIsolate();
  Local<Context> context = Context::New(isolate);
  Local<Number> number;
  int trap = 0;
  if (info[0]->ToInteger(context).ToLocal(&number)) {
    trap = number->Value();
  }
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
      toNative(info[1],isolate),
      toNative(info[2],isolate),
      toNative(info[3],isolate)
    );
    break;
  }
  int err = 0;
  if (r1 < 0) {
    err = errno;
  }
  Local<Array> res = Array::New(isolate, 3);
  res->Set(0, Integer::New(isolate, r1));
  res->Set(1, Integer::New(isolate, r2));
  res->Set(2, Integer::New(isolate, err));
  info.GetReturnValue().Set(res);
}

void Syscall6(const FunctionCallbackInfo<Value>& info) {
  Isolate* isolate = info.GetIsolate();
  Local<Context> context = Context::New(isolate);
  Local<Number> number;
  int trap = 0;
  if (info[0]->ToInteger(context).ToLocal(&number)) {
    trap = number->Value();
  }
  int r = syscall(
    trap,
    toNative(info[1],isolate),
    toNative(info[2],isolate),
    toNative(info[3],isolate),
    toNative(info[4],isolate),
    toNative(info[5],isolate),
    toNative(info[6],isolate)
  );
  int err = 0;
  if (r < 0) {
    err = errno;
  }
  Local<Array> res = Array::New(isolate, 3);
  res->Set(0, Integer::New(isolate, r));
  res->Set(1, Integer::New(isolate, 0));
  res->Set(2, Integer::New(isolate, err));
  info.GetReturnValue().Set(res);
}

void init(Handle<Object> target) {
  NODE_SET_METHOD(target, "Syscall", Syscall);
  NODE_SET_METHOD(target, "Syscall6", Syscall6);
}

NODE_MODULE(syscall, init);
