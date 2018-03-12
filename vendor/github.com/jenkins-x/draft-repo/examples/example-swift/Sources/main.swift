import Vapor

let drop = Droplet()
drop.get("/") { _ in
  return "Hello from Swift"
}
drop.run()

