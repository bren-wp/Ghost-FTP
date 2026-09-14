<?php
declare(strict_types=1);

namespace GhostFTP\Web;

use RuntimeException;

final class SftpTransport implements Transport
{
    private mixed $connection;
    private mixed $sftp;
    private string $root;

    public function __construct(array $profile)
    {
        if (!extension_loaded('ssh2')) throw new RuntimeException('SFTP requires the PHP ssh2 extension.');
        [, $ip] = Security::publicTarget((string)($profile['host'] ?? ''));
        $port=Security::port($profile['port']??null,22);
        $expected=Security::fingerprint((string)($profile['fingerprint']??''));
        $this->root=Security::remotePath((string)($profile['root']??'/'));
        $this->connection=@ssh2_connect($ip,$port);
        if(!$this->connection)throw new RuntimeException('Could not establish the SSH transport.');
        $raw=@ssh2_fingerprint($this->connection,SSH2_FINGERPRINT_SHA256|SSH2_FINGERPRINT_RAW);
        if(!is_string($raw)||$raw==='')throw new RuntimeException('Could not read the SFTP server host key.');
        $actual='SHA256:'.rtrim(base64_encode($raw),'=');
        if(!hash_equals($expected,$actual))throw new RuntimeException('SFTP host-key fingerprint mismatch.');
        $username=(string)($profile['username']??'');$password=(string)($profile['password']??'');
        try { if($username===''||!@ssh2_auth_password($this->connection,$username,$password))throw new RuntimeException('SFTP authentication failed.'); } finally { $password=''; $profile['password']=''; }
        $this->sftp=@ssh2_sftp($this->connection);if(!$this->sftp)throw new RuntimeException('SFTP subsystem is unavailable.');
    }

    public function list(string $path): array
    {
        $full=Security::remotePath($path,$this->root);$h=@opendir($this->uri($full));if(!$h)throw new RuntimeException('Could not list the remote directory.');$items=[];
        try{while(($name=readdir($h))!==false){if($name==='.'||$name==='..')continue;$remote=rtrim($full,'/').'/'.$name;$stat=@ssh2_sftp_lstat($this->sftp,$remote)?:[];$mode=(int)($stat['mode']??0);$dir=($mode&0040000)===0040000;$items[]=['name'=>$name,'type'=>$dir?'dir':'file','size'=>isset($stat['size'])?(int)$stat['size']:null,'modified'=>isset($stat['mtime'])?gmdate('c',(int)$stat['mtime']):null,'permissions'=>substr(sprintf('%o',$mode),-4)];}}finally{closedir($h);}usort($items,static fn($a,$b)=>$a['type']!==$b['type']?($a['type']==='dir'?-1:1):strnatcasecmp($a['name'],$b['name']));return $items;
    }
    public function mkdir(string $path): void { $path=Security::remotePath($path,$this->root);if(!@ssh2_sftp_mkdir($this->sftp,$path,0755,false))throw new RuntimeException('Could not create the directory.'); }
    public function rename(string $from,string $to):void{$from=Security::remotePath($from,$this->root);$to=Security::remotePath($to,$this->root);if(!@ssh2_sftp_rename($this->sftp,$from,$to))throw new RuntimeException('Rename failed.');}
    public function delete(string $path,bool $directory):void{$path=Security::remotePath($path,$this->root);$ok=$directory?@ssh2_sftp_rmdir($this->sftp,$path):@ssh2_sftp_unlink($this->sftp,$path);if(!$ok)throw new RuntimeException('Delete failed.');}
    public function chmod(string $path,int $mode):void{if($mode<0||$mode>0777)throw new RuntimeException('Invalid permission mode.');$path=Security::remotePath($path,$this->root);if(!@ssh2_sftp_chmod($this->sftp,$path,$mode))throw new RuntimeException('CHMOD failed.');}
    public function upload(string $localFile,string $remotePath):void{$remotePath=Security::remotePath($remotePath,$this->root);$size=filesize($localFile);if($size===false||$size>GHOSTFTP_WEB_MAX_UPLOAD_BYTES)throw new RuntimeException('Upload exceeds the configured limit.');$in=fopen($localFile,'rb');$out=@fopen($this->uri($remotePath),'wb');if(!is_resource($in)||!is_resource($out)){if(is_resource($in))fclose($in);if(is_resource($out))fclose($out);throw new RuntimeException('Could not start SFTP upload.');}try{$copied=stream_copy_to_stream($in,$out,GHOSTFTP_WEB_MAX_UPLOAD_BYTES+1);if($copied===false||$copied>$size)throw new RuntimeException('SFTP upload failed.');}finally{fclose($in);fclose($out);}}
    public function download(string $remotePath,string $localFile,int $maxBytes):int{$remotePath=Security::remotePath($remotePath,$this->root);$in=@fopen($this->uri($remotePath),'rb');$out=fopen($localFile,'w+b');if(!is_resource($in)||!is_resource($out)){if(is_resource($in))fclose($in);if(is_resource($out))fclose($out);throw new RuntimeException('Could not start SFTP download.');}try{$total=0;while(!feof($in)){$chunk=fread($in,min(65536,$maxBytes-$total+1));if($chunk===false)throw new RuntimeException('SFTP download failed.');$total+=strlen($chunk);if($total>$maxBytes)throw new RuntimeException('Download exceeds the configured limit.');if($chunk!==''&&fwrite($out,$chunk)!==strlen($chunk))throw new RuntimeException('Could not write the temporary download.');}return $total;}catch(\Throwable $e){ftruncate($out,0);throw $e;}finally{fclose($in);fclose($out);}}
    public function read(string $remotePath,int $maxBytes):string{$tmp=tempnam(sys_get_temp_dir(),'gftp-sftp-read-');if($tmp===false)throw new RuntimeException('Could not create a temporary file.');try{$this->download($remotePath,$tmp,$maxBytes);$data=file_get_contents($tmp);if(!is_string($data)||str_contains(substr($data,0,8192),"\0"))throw new RuntimeException('Binary files cannot be edited in Web FTP.');return $data;}finally{@unlink($tmp);}}
    public function write(string $remotePath,string $content):void{if(strlen($content)>GHOSTFTP_WEB_MAX_EDIT_BYTES||str_contains(substr($content,0,8192),"\0"))throw new RuntimeException('Editor content is invalid or too large.');$remotePath=Security::remotePath($remotePath,$this->root);$fp=@fopen($this->uri($remotePath),'wb');if(!is_resource($fp))throw new RuntimeException('Could not open the remote file for writing.');try{$remaining=$content;while($remaining!==''){$written=fwrite($fp,$remaining);if($written===false||$written===0)throw new RuntimeException('Remote write failed.');$remaining=substr($remaining,$written);}}finally{fclose($fp);}}
    public function close():void{$this->sftp=null;$this->connection=null;}
    private function uri(string $path):string{return 'ssh2.sftp://'.intval($this->sftp).$path;}
}
