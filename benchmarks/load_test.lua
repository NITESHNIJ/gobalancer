-- wrk2 Lua script for sustained load testing of gobalancer
-- Usage: wrk2 -t4 -c100 -d60s -R10000 --latency -s benchmarks/load_test.lua http://localhost:8080

local counter = 0
local paths = {"/", "/api/users", "/api/orders", "/static/style.css", "/health"}

request = function()
  counter = counter + 1
  local path = paths[(counter % #paths) + 1]
  return wrk.format("GET", path, {
    ["X-Request-ID"] = tostring(counter),
    ["Accept"] = "application/json",
  })
end

response = function(status, headers, body)
  if status >= 500 then
    io.write("5xx: " .. status .. "\n")
  end
end

done = function(summary, latency, requests)
  io.write("-----\n")
  io.write(string.format("Requests/sec: %.2f\n", summary.requests / (summary.duration / 1e6)))
  io.write(string.format("Transfer/sec: %.2f MB\n", summary.bytes / (summary.duration / 1e6) / 1e6))
  for _, p in pairs({50, 75, 90, 99, 99.9}) do
    local lat = latency:percentile(p)
    io.write(string.format("p%.1f latency: %d μs\n", p, lat))
  end
end
