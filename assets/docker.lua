docker = {}

function docker.build(path, args)
  -- construct initial "docker build" command
  cmd = {"docker", "build"}

  -- append arguments
  if type(args) == "table" then
    for i=1,#args do
      table.insert(cmd, args[i])
    end
  end

  -- append package to be build
  table.insert(cmd, path)
  
  exec(table.unpack(cmd))
end
