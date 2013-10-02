#include <node.h>
#include <v8.h>
#include <unistd.h>
#include <iostream>

using namespace v8;

#define SYS_FORK 2
#define SYS_PIPE 42

const char* ABC = "abc";

size_t toNative(Local<Value> value) {
  if (value.IsEmpty()) {
    return 0;
  }
  if (value->IsArrayBufferView()) {
    Local<ArrayBufferView> view = Local<ArrayBufferView>::Cast(value);
    return (size_t)view->BaseAddress();
  }
  if (value->IsArray()) {
    Local<Array> array = Local<Array>::Cast(value);
    size_t* native = (size_t*)malloc(array->Length() * sizeof(size_t)); // TODO memory leak
    for (uint32_t i = 0; i < array->Length(); i++) {
      native[i] = toNative(array->CloneElementAt(i));
    }
    return (size_t)native;
  }
  return value->ToInteger()->Value();
}

void Syscall(const FunctionCallbackInfo<Value>& info) {
  int trap = info[0]->ToInteger()->Value();
  int r1 = 0, r2 = 0;
  switch (trap) {
  case SYS_FORK:
    r1 = fork();
    break;
  case SYS_PIPE:
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
      toNative(info[1]),
      toNative(info[2]),
      toNative(info[3])
    );
    break;
  }
  int err = 0;
  if (r1 < 0) {
    err = errno;
  }
  Local<Array> res = Array::New(3);
  res->Set(0, Integer::New(r1));
  res->Set(1, Integer::New(r2));
  res->Set(2, Integer::New(err));
  info.GetReturnValue().Set(res);
}

void Syscall6(const FunctionCallbackInfo<Value>& info) {
  int r = syscall(
    info[0]->ToInteger()->Value(),
    toNative(info[1]),
    toNative(info[2]),
    toNative(info[3]),
    toNative(info[4]),
    toNative(info[5]),
    toNative(info[6])
  );
  int err = 0;
  if (r < 0) {
    err = errno;
  }
  Local<Array> res = Array::New(3);
  res->Set(0, Integer::New(r));
  res->Set(1, Integer::New(0));
  res->Set(2, Integer::New(err));
  info.GetReturnValue().Set(res);
}

void init(Handle<Object> target) {
  NODE_SET_METHOD(target, "Syscall", Syscall);
  NODE_SET_METHOD(target, "Syscall6", Syscall6);
}

NODE_MODULE(syscall, init);