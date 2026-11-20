//go:build linux && arm && cgo

#include "_cgo_export.h"

// InkViewMain wants a plain C function pointer; forward to the Go handler.
int main_handler(int t, int p1, int p2) {
    return goMainHandler(t, p1, p2);
}

// OpenKeyboard wants a plain C callback; forward to the Go handler.
void c_keyboard_handler(char *text) {
    goKeyboardHandler(text);
}

// SetHardTimer wants a plain C callback; forward to the Go handler.
void c_timer_handler() {
    goTimerHandler();
}
