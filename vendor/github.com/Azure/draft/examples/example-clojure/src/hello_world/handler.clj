(ns hello-world.handler
  (:require [compojure.core :refer :all]
            [compojure.route :as route]
            [ring.middleware.defaults :refer [wrap-defaults site-defaults]]))

(defn hello [request]
  "Hello World")

(defroutes app-routes
  (GET "/" [] hello)
  (route/not-found "Not Found"))

(def app
  (wrap-defaults app-routes site-defaults))
