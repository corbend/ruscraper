(function(global, angular) {
	angular.module('App').service('websocket', function() {

		var socket;
		var messageHandler;

		function connect(port) {
			var url = "ws://localhost:" + (port || 3000) + "/ws";
			socket = new WebSocket(url);

			socket.onmessage = function(msg){
				if (msg.action == "parse_start") {
					messageHandler(msg.action, msg.payload);
				} else if (msg.action == "parse_complete") {
					messageHandler(msg.action, msg.payload);
				} else if (msg.action == "new_update") {
					messageHandler(msg.action, msg.payload);
				}
			}

			socket.onopen = function(){
				console.log("connected to websocket")
			}
		}

		connect(3000);

		return {
			setMessageHandler: function(hdl) {
				messageHandler = hdl;
			},
			connect: connect,
			socket: socket
		}
	})
})(window, angular)