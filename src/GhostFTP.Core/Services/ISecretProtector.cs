namespace GhostFTP.Core.Services;

public interface ISecretProtector
{
    string Protect(string plaintext);
    string Unprotect(string protectedText);
}
