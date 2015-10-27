function target(name, dependencies, fn)
  assert(type(name) == "string", "invalid target name type: " .. type(name))

  -- fn can be passed in as second arg.
  if type(dependencies) == "function" then
    fn = dependencies
    dependencies = {}
  end

  __bake_begin_target(name, dependencies)
  if type(fn) == "function" then
    fn()
  end
  __bake_end_target(name)
end

function title(text)
  __bake_set_title(text)
end

