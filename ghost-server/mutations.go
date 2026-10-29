package ghostserver

import (
 "errors"
 "os"
 "path/filepath"
 "strings"
)

// safeMutationPath resolves a real existing path relative to os.Root.
// The final filesystem entry must be a normal file for DELE/RNFR.
func (s *Server) safeMutationPath(cwd,operand string,directory bool)(string,error) {
 if operand=="" {return "",errors.New("missing mutation operand")}
 _,virtual,err:=s.resolve(cwd,operand)
 if err!=nil{return "",err}
 relative:=rootedPath(virtual)
 if relative=="."||filepath.IsAbs(relative){return "",errors.New("cannot mutate filesystem root")}
 for _,part:=range strings.Split(virtual,"/") {
  if isReservedInternalName(part){return "",errors.New("private internal entry is not mutable")}
 }
 info,err:=s.uploadRoot.Lstat(relative)
 if err!=nil{return "",err}
 if directory {
  if !info.IsDir(){return "",errors.New("target is not a directory")}
 } else {
  if !info.Mode().IsRegular(){return "",errors.New("target is not a regular file")}
 }
 return relative,nil
}

func (s *Server) mutationAllowed(c *client) bool {
 if !s.allowDestructive { _=c.reply(550,"Destructive operations disabled by server policy");return false }
 if !c.private { _=c.reply(534,"Private TLS data protection required");return false }
 return true
}

func (s *Server) deleteFile(c *client,operand string) {
 if !s.mutationAllowed(c){return}
 relative,err:=s.safeMutationPath(c.cwd,operand,false)
 if err!=nil{_ =c.reply(550,"File unavailable or unsafe");return}
 if err=s.uploadRoot.Remove(relative);err!=nil{_=c.reply(550,"File could not be removed");return}
 _=c.reply(250,"File removed")
}

func (s *Server) removeDirectory(c *client,operand string) {
 if !s.mutationAllowed(c){return}
 relative,err:=s.safeMutationPath(c.cwd,operand,true)
 if err!=nil{_=c.reply(550,"Directory unavailable or unsafe");return}
 // os.Root.Remove rejects nonempty directories on supported filesystems.
 if err=s.uploadRoot.Remove(relative);err!=nil{_=c.reply(550,"Directory not empty or cannot be removed");return}
 _=c.reply(250,"Empty directory removed")
}

func (s *Server) startRename(c *client,operand string) {
 c.renameSource=""
 if !s.mutationAllowed(c){return}
 relative,err:=s.safeMutationPath(c.cwd,operand,false)
 if err!=nil{_=c.reply(550,"Only existing regular files may be renamed");return}
 c.renameSource=relative
 _=c.reply(350,"File selected; send RNTO")
}

func (s *Server) finishRename(c *client,operand string) {
 source:=c.renameSource
 c.renameSource=""
 if !s.mutationAllowed(c){return}
 if source==""{_=c.reply(503,"RNFR is required before RNTO");return}
 destination,err:=s.createDestination(c.cwd,operand)
 if err!=nil{_=c.reply(550,"Rename target unavailable or already exists");return}
 target,err:=filepath.Rel(s.root,destination)
 if err!=nil||target==".."||strings.HasPrefix(target,".."+string(os.PathSeparator)){
  _=c.reply(550,"Rename target outside root");return
 }
 // Revalidate the source before mutation, including against symlink swaps.
 info,err:=s.uploadRoot.Lstat(source)
 if err!=nil||!info.Mode().IsRegular(){_=c.reply(550,"Rename source changed or unavailable");return}
 // An atomic hardlink cannot replace an existing destination name.
 if err=s.uploadRoot.Link(source,target);err!=nil{
  _=c.reply(550,"Target exists or filesystem cannot safely rename");return
 }
 if err=s.uploadRoot.Remove(source);err!=nil{
  _=c.reply(451,"File linked but original could not be removed");return
 }
 _=c.reply(250,"File renamed without overwrite")
}
