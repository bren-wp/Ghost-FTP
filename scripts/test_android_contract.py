#!/usr/bin/env python3

class AndroidContractTests(unittest.TestCase):
    def read(self, rel: str) -> str:
        return (ROOT / rel).read_text(encoding="utf-8")

        for marker in (
            'command("AUTH TLS")',
            'parameters.setEndpointIdentificationAlgorithm("HTTPS")',
            'command("PBSZ 0")',
            'command("PROT P")',

if __name__ == "__main__":
    unittest.main()
