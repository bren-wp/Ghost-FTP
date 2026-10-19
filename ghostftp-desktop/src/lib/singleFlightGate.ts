/** A synchronous gate across await points. Failed launches must release it. */
export function createSingleFlightGate() {
  let pending = false;
  return {
    tryAcquire(): boolean {
      if (pending) return false;
      pending = true;
      return true;
    },
    release(): void {
      pending = false;
    },
  };
}
