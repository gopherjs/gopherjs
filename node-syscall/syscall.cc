#include <node.h>
#include <v8.h>
#include <unistd.h>

using namespace v8;

size_t toNative(v8::Local<v8::Value> value) {
  if (value->IsArrayBufferView()) {
    Local<ArrayBufferView> view = Local<ArrayBufferView>::Cast(value);
    return size_t(view->BaseAddress());
  }
  return value->ToInteger()->Value();
}

void Syscall(const FunctionCallbackInfo<Value>& info) {
  int r = syscall(
    info[0]->ToInteger()->Value(),
    toNative(info[1]),
    toNative(info[2]),
    toNative(info[3])
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
}

NODE_MODULE(syscall, init);