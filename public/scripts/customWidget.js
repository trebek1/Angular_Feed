/**
 * @ngdoc module
 * @name Heelix.CustomWidget
 *
 * @description
 *
 *
 */
angular.module('Heelix.CustomWidget', [
/**
 * @ngdoc controller
 * @name customWidgetController
 *
 * @description
 *
 *
 */
]).controller('customWidgetController', ['$scope', '$http', '$interval', function($scope, $http, $interval) {
   var documents = []
   // Pull data retrieves the data from the API and unshifts the top 20 entries from the Latest News section to an array  
   // Unshift places the new news first on the feed 
    function pullData(){
        $http.post("http://localhost:8081/api/all_entity_info").success(function(response){
            for(var i =0; i<20; i++){
                documents.unshift(response.LatestNews[i])                          
            }   
        });
    }
    // call the function to pull data from the API
    pullData(); 
    // Use interval to fire the function every 10 seconds, (10,000 milliseconds)
    $interval(function(){

        pullData()
        // if the length of the array being shown is more than 50, we pop off the oldest stories (last in array)
        while(documents.length-1 > 50){
            documents.pop()
        }
    },10000);

     // Save the documents array to access in the view 
     $scope.documents = documents; 
     // range for ng-repeat for remaining 4 entries after initial
     $scope.range = [1,2,3,4]

}]);




