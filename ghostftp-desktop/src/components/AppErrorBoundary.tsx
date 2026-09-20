import React from "react";

interface State {
  failed: boolean;
  message: string;
}

function safeDiagnostic(error: unknown): string {
  const raw = error instanceof Error ? error.message : "Unexpected UI error";
  return raw
    .replace(/[A-Za-z]:\\Users\\[^\\\s]+/gi, "C:\\Users\\<user>")
    .replace(/\/home\/[^/\s]+/g, "/home/<user>")
    .replace(/([?&](?:token|code|password|passphrase|secret|key)=)[^&\s]+/gi, "$1<redacted>")
    .slice(0, 320);
}

/**
 * Last-resort UI containment for an unexpected render failure. Network,
 * filesystem and protocol errors are handled closer to their operations; this
 * boundary prevents one bad view from turning the entire desktop window blank.
 */
export class AppErrorBoundary extends React.Component<React.PropsWithChildren, State> {
  state: State = { failed: false, message: "" };

  static getDerivedStateFromError(error: unknown): State {
    return { failed: true, message: safeDiagnostic(error) };
  }

  componentDidCatch(error: unknown) {
    console.error("Ghost FTP UI failure:", safeDiagnostic(error));
  }

  private recover = () => {
    this.setState({ failed: false, message: "" });
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
          {this.state.message && (
            <details className="mt-4 rounded-lg border border-[#214d70] bg-[#041526] p-3 text-xs text-[#a9cce2]" open>
              <summary className="cursor-pointer font-medium text-[#dff4ff]">Technical details</summary>
              <code className="mt-2 block break-words whitespace-pre-wrap">{this.state.message}</code>
            </details>
          )}
          <button className="ghost-primary-button mt-5" onClick={this.recover}>Reload Ghost FTP</button>
        </section>
      </main>
    );
  }
}
