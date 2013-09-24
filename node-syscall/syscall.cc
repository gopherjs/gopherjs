#include <node.h>
#include <v8.h>
#include <node_buffer.h>
#include <unistd.h>
#include <iostream>

using namespace v8;

void* toNative(v8::Local<v8::Value> value) {
  if (node::Buffer::HasInstance(value)) {
    return node::Buffer::Data(value);
  }
  return (void*) value->ToInteger()->Value();
}

Handle<Value> Syscall(const Arguments& args) {
  HandleScope scope;
  syscall(
    args[0]->ToInteger()->Value(),
    toNative(args[1]),
    toNative(args[2]),
    toNative(args[3])
  );
  return scope.Close(String::New("world"));
}

void init(Handle<Object> target) {
  NODE_SET_METHOD(target, "Syscall", Syscall);
}

NODE_MODULE(syscall, init);