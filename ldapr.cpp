#include <atomic>
#include <chrono>
#include <cstdint>
#include <iomanip>
#include <iostream>

constexpr size_t ITERATIONS = 50'000'000;

// Two separate cache lines
struct alignas(64) Line {
  std::atomic<uint64_t> value{0};
};
Line lineA;
Line lineB;

//------------------------------------------------------------------------------
// Pattern: Store-release to A, then Acquire-load from B
// LDAR: Must wait for store to A to drain before loading B
// LDAPR: Can load B immediately (different address)
//------------------------------------------------------------------------------
template <std::memory_order order> uint64_t stlr_load() {
  uint64_t sum = 0;
  for (size_t i = 0; i < ITERATIONS; ++i) {
    // Store-release to line A
    lineA.value.store(i, std::memory_order_release);
    // Acquire load from line B
    sum += lineB.value.load(order);
  }
  return sum;
}

template <typename Func> double benchmark(Func &&f, const char *name) {
  volatile static uint64_t sink = 0;
  // Warmup
  sink = f();
  auto start = std::chrono::high_resolution_clock::now();
  sink = f();
  auto end = std::chrono::high_resolution_clock::now();
  double ms = std::chrono::duration<double, std::milli>(end - start).count();
  double ns_per_op = (ms * 1'000'000.0) / ITERATIONS;
  std::cout << "  " << name << ": " << ns_per_op << " ns/op\n";
  return ns_per_op;
}

int main(int argc, char *argv[]) {
  if (argc != 2) {
    std::cout << "Usage: " << argv[0] << " [N iterations]\n";
    return 1;
  }
  int n = std::atoi(argv[1]);
  double t_ldapr = 0.0;
  double t_ldar = 0.0;
  double t_ldr = 0.0;
  for (int i = 0; i < n; i++) {
    t_ldapr += benchmark(stlr_load<std::memory_order_acquire>, "LDAPR    ");
    t_ldar += benchmark(stlr_load<std::memory_order_seq_cst>, "LDAR     ");
    t_ldr += benchmark(stlr_load<std::memory_order_relaxed>, "LDR      ");
  }
  double d_ldapr = (((t_ldar - t_ldapr) / t_ldar)) * 100.0;
  double d_ldr = (((t_ldar - t_ldr) / t_ldar)) * 100.0;
  std::cout << "\nResult: acquire (ldapr) is " << std::fixed
            << std::setprecision(1) << d_ldapr
            << "% faster than ldar and relaxed (ldr) is " << std::fixed
            << std::setprecision(1) << d_ldr << "% faster than seq_cst(ldar)\n";
}
