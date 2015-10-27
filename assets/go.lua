go = {}

function go.build(pkg, args)
  -- construct initial "go build" command
  cmd = {"go", "build"}

  -- append arguments
  if type(args) == "table" then
    for i=1,#args do
      table.insert(cmd, args[i])
    end
  end

  -- append package to be build
  table.insert(cmd, pkg)
  
  exec(table.unpack(cmd))
end
