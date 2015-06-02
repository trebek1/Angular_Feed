/**
 * @ngdoc module
 * @name Heelix
 *
 * @description
 *
 * # Heelix (Our App File)
 *
 * This file bootstraps our Angular appliation; for the purposes of this demo, we created a simple
 * single controller-less module that contains a single Widget directive. The Widget directive
 * provides the general wrapper for the demo and some basic convenience methods.
 *
 * You can modify this file if needed, though you should assume that the heelixWidget directive is
 * used by many other widgets app-wide. If you do make any edits to it, they should be "safe" for
 * any other modules that might share this wrapper.
 *
 */
angular.module('Heelix', [

    // pull in our custom widget module, loaded as a dependency automatically
    'Heelix.CustomWidget'

// our general Widget wrapper constructor
]).directive('heelixWidget', [function() {
    
    return {
        
        // widget wrapper restricted to element-level compilation
        restrict : 'E',

        // load our Widget template for providing the common shell
        templateUrl : 'templates/doNotEditWidget.html',
        replace     : true,

        /**
         * @ngdoc method
         * @name link
         *
         * @param {object} scope Our widget scope object, local to the directive
         * @param {object} el Our widget element
         *
         * @description
         *
         * As per the Angular directive spec, our link function provides the functional controls
         * and interaction rules of our directive when it's compiled.
         *
         * This directive is shared by all Heelix widgets and is generic by design. Any edits made
         * here will be shared by any down-stream widgets.
         *
         */
        link: function(scope, el) {

            var
                // in production, this would be a randomly generated string. for demo purposes,
                // it's hard-coded to make things a little simpler.
                widgetId = 'widget12345';


            /**
             * @ngdoc method
             * @name showWidgetMenu
             *
             * @description
             *
             * This is a stubbed method for controlling our Widget menu; it's not expected to be
             * used for this demo unless you need it for something.
             *
             */
            scope.showWidgetMenu = function() {
                console.log('Widget menu functionality is stubbed unless needed.');
            };

            /**
             * @ngdoc method
             * @name closeWidget
             *
             * @description
             *
             * This is a stubbed method for dismissing our Widget. It provides a simple hook for
             * destroying down-stream widgets.
             *
             */
            scope.dismissWidget = function() {
                scope.$broadcast('widget:dismissed', widgetId);
                console.log('Aside from the custom event, this method is otherwise stubbed.');
            }

            // hook for down-stream modules
            scope.$broadcast('widget:rendered', widgetId);

            console.log('Widget rendered:', widgetId);
        }
    };

}]);