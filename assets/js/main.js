(function(global, angular) {

	var app = angular.module('App', []);
	app.config(function($interpolateProvider) {
    	$interpolateProvider.startSymbol('//');
    	$interpolateProvider.endSymbol('//');
  	});
})(window, angular)