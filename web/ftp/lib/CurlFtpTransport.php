<?php
declare(strict_types=1);

namespace GhostFTP\Web;

use RuntimeException;

final class CurlFtpTransport implements Transport
{
    private string $host;
    private string $ip;
    private int $port;
    private string $protocol;
    private string $username;
    private string $password;
    private string $root;

    public function __construct(array $profile)
    {
        if (!extension_loaded('curl')) throw new RuntimeException('The PHP cURL extension is required for FTP/FTPS.');
        [$this->host, $this->ip] = Security::publicTarget((string)($profile['host'] ?? ''));
        $this->protocol = (string)($profile['protocol'] ?? 'ftp');
        if (!in_array($this->protocol, ['ftp', 'ftps'], true)) throw new RuntimeException('Unsupported FTP protocol.');
        $this->port = Security::port($profile['port'] ?? null, 21);
        $this->username = (string)($profile['username'] ?? '');
        $this->password = (string)($profile['password'] ?? '');
        $this->root = Security::remotePath((string)($profile['root'] ?? '/'));
        if ($this->username === '' || strlen($this->username) > 256) throw new RuntimeException('Username is required.');
    }

    public function list(string $path): array
    {
        $path = Security::remotePath($path, $this->root);
        $raw = $this->request($path, ['custom' => 'MLSD', 'return' => true]);
        $items = $this->parseMlsd((string)$raw);
        if (!$items) {
            $raw = $this->request($path, ['custom' => 'LIST', 'return' => true]);
            $items = $this->parseList((string)$raw);
        }
        usort($items, static fn(array $a, array $b): int => $a['type'] !== $b['type'] ? ($a['type'] === 'dir' ? -1 : 1) : strnatcasecmp($a['name'], $b['name']));
        return $items;
    }

    public function mkdir(string $path): void { $path = Security::remotePath($path, $this->root); $this->request($this->root, ['quote' => ['MKD ' . $this->commandPath($path)]]); }
    public function rename(string $from, string $to): void { $from=Security::remotePath($from,$this->root);$to=Security::remotePath($to,$this->root);$this->request($this->root,['quote'=>['RNFR '.$this->commandPath($from),'RNTO '.$this->commandPath($to)]]); }
    public function delete(string $path, bool $directory): void { $path=Security::remotePath($path,$this->root);$this->request($this->root,['quote'=>[($directory?'RMD ':'DELE ').$this->commandPath($path)]]); }
    public function chmod(string $path, int $mode): void { if ($mode < 0 || $mode > 0777) throw new RuntimeException('Invalid permission mode.');$path=Security::remotePath($path,$this->root);$this->request($this->root,['quote'=>['SITE CHMOD '.sprintf('%04o',$mode).' '.$this->commandPath($path)]]); }

    public function upload(string $localFile, string $remotePath): void
    {
        $remotePath = Security::remotePath($remotePath, $this->root);
        $size = filesize($localFile);
        if ($size === false || $size > GHOSTFTP_WEB_MAX_UPLOAD_BYTES) throw new RuntimeException('Upload exceeds the configured limit.');
        $fp = fopen($localFile, 'rb');
        if (!is_resource($fp)) throw new RuntimeException('Could not open upload.');
        try { $this->request($remotePath, ['upload' => $fp, 'size' => $size]); } finally { fclose($fp); }
    }

    public function download(string $remotePath, string $localFile, int $maxBytes): int
    {
        $remotePath = Security::remotePath($remotePath, $this->root);
        $fp = fopen($localFile, 'w+b');
        if (!is_resource($fp)) throw new RuntimeException('Could not create the temporary download file.');
        $received = 0;
        $ch = $this->handle($remotePath);
        curl_setopt($ch, CURLOPT_FILE, $fp);
        curl_setopt($ch, CURLOPT_NOPROGRESS, false);
        curl_setopt($ch, CURLOPT_XFERINFOFUNCTION, static function ($resource, float $downloadTotal, float $downloadNow) use ($maxBytes, &$received): int { $received=(int)$downloadNow; return $downloadNow > $maxBytes ? 1 : 0; });
        try { $this->exec($ch); } catch (RuntimeException $e) { ftruncate($fp, 0); throw $e; } finally { curl_close($ch); fclose($fp); }
        $actual = filesize($localFile);
        if ($actual === false || $actual > $maxBytes) { @unlink($localFile); throw new RuntimeException('Download exceeded the configured limit.'); }
        return (int)$actual;
    }

    public function read(string $remotePath, int $maxBytes): string
    {
        $tmp = tempnam(sys_get_temp_dir(), 'gftp-read-');
        if ($tmp === false) throw new RuntimeException('Could not create a temporary file.');
        try { $this->download($remotePath, $tmp, $maxBytes); $data=file_get_contents($tmp); if(!is_string($data)||str_contains(substr($data,0,8192),"\0")) throw new RuntimeException('Binary files cannot be edited in Web FTP.'); return $data; } finally { @unlink($tmp); }
    }

    public function write(string $remotePath, string $content): void
    {
        if (strlen($content) > GHOSTFTP_WEB_MAX_EDIT_BYTES || str_contains(substr($content,0,8192),"\0")) throw new RuntimeException('Editor content is invalid or too large.');
        $tmp=tempnam(sys_get_temp_dir(),'gftp-write-');if($tmp===false)throw new RuntimeException('Could not create a temporary file.');
        try { if(file_put_contents($tmp,$content,LOCK_EX)===false)throw new RuntimeException('Could not prepare editor content.');$this->upload($tmp,$remotePath); } finally { @unlink($tmp); }
    }

    public function close(): void { $this->password=''; }

    private function request(string $path, array $options): string|bool
    {
        $ch=$this->handle($path);
        if(isset($options['custom']))curl_setopt($ch,CURLOPT_CUSTOMREQUEST,$options['custom']);
        if(isset($options['quote']))curl_setopt($ch,CURLOPT_QUOTE,$options['quote']);
        if(isset($options['upload'])){curl_setopt($ch,CURLOPT_UPLOAD,true);curl_setopt($ch,CURLOPT_INFILE,$options['upload']);curl_setopt($ch,CURLOPT_INFILESIZE,(int)$options['size']);}
        if(($options['return']??false)===true)curl_setopt($ch,CURLOPT_RETURNTRANSFER,true);else curl_setopt($ch,CURLOPT_NOBODY,true);
        try{return $this->exec($ch);}finally{curl_close($ch);}
    }

    private function handle(string $path): mixed
    {
        $encoded = implode('/', array_map('rawurlencode', array_filter(explode('/', ltrim($path,'/')), static fn($v)=>$v!=='')));
        $url='ftp://'.$this->host.':'.$this->port.'/'.($encoded!==''?$encoded:'');
        $ch=curl_init($url);if($ch===false)throw new RuntimeException('Could not initialize transfer.');
        $resolved = str_contains($this->ip, ':') ? '['.$this->ip.']' : $this->ip;
        curl_setopt_array($ch,[CURLOPT_USERPWD=>$this->username.':'.$this->password,CURLOPT_CONNECTTIMEOUT=>GHOSTFTP_WEB_CONNECT_TIMEOUT,CURLOPT_TIMEOUT=>GHOSTFTP_WEB_OPERATION_TIMEOUT,CURLOPT_FTP_USE_EPSV=>true,CURLOPT_PROTOCOLS=>CURLPROTO_FTP,CURLOPT_REDIR_PROTOCOLS=>0,CURLOPT_FOLLOWLOCATION=>false,CURLOPT_RESOLVE=>[$this->host.':'.$this->port.':'.$resolved]]);
        if($this->protocol==='ftps'){curl_setopt($ch,CURLOPT_USE_SSL,CURLUSESSL_ALL);curl_setopt($ch,CURLOPT_SSL_VERIFYPEER,true);curl_setopt($ch,CURLOPT_SSL_VERIFYHOST,2);}
        return $ch;
    }

    private function exec(mixed $ch): string|bool
    {
        $result=curl_exec($ch);if($result===false){$message=curl_error($ch);throw new RuntimeException('FTP operation failed'.($message!==''?': '.$message:'.'));}
        $code=(int)curl_getinfo($ch,CURLINFO_RESPONSE_CODE);if($code>=400)throw new RuntimeException('FTP server rejected the operation.');return $result;
    }

    private function commandPath(string $path): string { if(preg_match('/[\r\n]/',$path))throw new RuntimeException('Unsafe FTP command path.');return $path; }

    private function parseMlsd(string $raw): array
    {
        $out=[];foreach(preg_split('/\r?\n/',$raw)?:[] as $line){$line=trim($line);if($line===''||!str_contains($line,' '))continue;[$facts,$name]=explode(' ',$line,2);$name=trim($name);if($name===''||$name==='.'||$name==='..')continue;$map=[];foreach(explode(';',$facts) as $fact){if(!str_contains($fact,'='))continue;[$k,$v]=explode('=',$fact,2);$map[strtolower($k)]=$v;}$type=strtolower($map['type']??'file');if(in_array($type,['cdir','pdir'],true))continue;$out[]=['name'=>$name,'type'=>$type==='dir'?'dir':'file','size'=>isset($map['size'])?(int)$map['size']:null,'modified'=>$map['modify']??null,'permissions'=>$map['unix.mode']??''];}return $out;
    }

    private function parseList(string $raw): array
    {
        $out=[];foreach(preg_split('/\r?\n/',$raw)?:[] as $line){if(!preg_match('/^([dl-])[rwxstST-]{9}\s+\d+\s+\S+\s+\S+\s+(\d+)\s+\w+\s+\d+\s+[\d:]+\s+(.+)$/',$line,$m))continue;$name=trim($m[3]);if($name===''||$name==='.'||$name==='..')continue;$out[]=['name'=>$name,'type'=>$m[1]==='d'?'dir':'file','size'=>(int)$m[2],'modified'=>null,'permissions'=>''];}return $out;
    }
}
