const SECRET_QUERY =
  /([?&](?:token|code|password|passphrase|secret|api[_-]?key|access[_-]?token|refresh[_-]?token)=)[^&\s]+/gi;
const SECRET_ASSIGNMENT =
  /\b((?:password|passphrase|secret|token|api[_-]?key|access[_-]?token|refresh[_-]?token|aws_secret_access_key|pgpassword)\s*=\s*)(?:"[^"]*"|'[^']*'|[^\s;]+)/gi;
const SECRET_FLAG =
  /(\s--?(?:password|passphrase|secret|token|api-key|apikey|access-token|refresh-token)(?:=|\s+))(?:"[^"]*"|'[^']*'|\S+)/gi;
const AUTH_HEADER =
  /\b((?:authorization|proxy-authorization)\s*:\s*)(?:bearer|basic)?\s*[^\s,;]+/gi;
const JSON_SECRET =
  /(["'](?:password|passphrase|secret|token|api[_-]?key|access[_-]?token|refresh[_-]?token)["']\s*:\s*)["'][^"']*["']/gi;
const BEARER_TOKEN = /\b(Bearer\s+)[A-Za-z0-9._~+/=-]+/gi;
const URL_PASSWORD =
  /([a-z][a-z0-9+.-]*:\/\/[^:\s/@]+:)[^@\s/]+@/gi;
const PRIVATE_KEY_BLOCK =
  /-----BEGIN [^-\r\n]*PRIVATE KEY-----[\s\S]*?-----END [^-\r\n]*PRIVATE KEY-----/gi;

/**
 * Redact common credentials before diagnostic text reaches the UI, persisted
 * state or production logs. This is deliberately conservative: hiding a value
 * that only looks secret is safer than exposing a real token.
 */
export function redactSensitiveText(value: unknown, maxLength?: number): string {
  const raw =
    typeof value === "string"
      ? value
      : value instanceof Error
        ? value.message
        : String(value ?? "");

  const redacted = raw
    .replace(/[A-Za-z]:\\Users\\[^\\\s]+/gi, "C:\\Users\\<user>")
    .replace(/\/home\/[^/\s]+/g, "/home/<user>")
    .replace(URL_PASSWORD, "$1<redacted>@")
    .replace(SECRET_QUERY, "$1<redacted>")
    .replace(SECRET_ASSIGNMENT, "$1<redacted>")
    .replace(SECRET_FLAG, "$1<redacted>")
    .replace(AUTH_HEADER, "$1<redacted>")
    .replace(JSON_SECRET, '$1"<redacted>"')
    .replace(BEARER_TOKEN, "$1<redacted>")
    .replace(PRIVATE_KEY_BLOCK, "<redacted private key>");

  return typeof maxLength === "number" ? redacted.slice(0, maxLength) : redacted;
}
