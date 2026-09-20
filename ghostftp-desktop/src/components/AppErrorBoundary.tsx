import React from "react";

interface State {
  failed: boolean;
}

/**
 * Last-resort UI containment for an unexpected render failure. Network,
 * filesystem and protocol errors are handled closer to their operations; this
 * boundary prevents one bad view from turning the entire desktop window blank.
 */
export class AppErrorBoundary extends React.Component<React.PropsWithChildren, State> {
  state: State = { failed: false };

  static getDerivedStateFromError(): State {
    return { failed: true };
  }

  componentDidCatch(error: unknown) {
    const message = error instanceof Error ? error.message : "Unexpected UI error";
    // Never serialize application state/credentials into diagnostics.
    console.error("Ghost FTP UI failure:", message);
  }

  private recover = () => {
    this.setState({ failed: false });
    window.location.reload();
  };

  render() {
    if (!this.state.failed) return this.props.children;
    return (
      <main className="min-h-screen bg-[#041526] p-6 text-[#eaf6ff]" role="alert">
        <section className="mx-auto mt-16 max-w-xl rounded-xl border border-[#245b84] bg-[#071d32] p-6 shadow-2xl">
          <h1 className="text-xl font-semibold">Ghost FTP recovered from an interface error</h1>
          <p className="mt-3 text-sm text-[#9fc3db]">
            Your files and server data were not modified by this interface failure. Reload the application to restore the workspace.
          </p>
          <button className="ghost-primary-button mt-5" onClick={this.recover}>Reload Ghost FTP</button>
        </section>
      </main>
    );
  }
}
