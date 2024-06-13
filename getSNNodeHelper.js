var GetSNNodeHelper = Class.create();
GetSNNodeHelper.prototype = Object.extendsObject(AbstractAjaxProcessor, {

    //ServiceNow Adaptation of this plugin - https://github.com/arnoudkooi/ServiceNow-Utils/blob/master/background.js
    getNodeBigIP: function() {
        try {
            var url = gs.getProperty('glide.servlet.uri');
            var nodeId = '',
                nodeName = '';
            var encodedPort = '';
            var encodedIP = 0;

            var getNode = new GlideRecord('sys_cluster_state');
            getNode.addEncodedQuery('sys_id=' + this.getParameter('sysparm_nodeID'));
            getNode.setLimit(1);
            getNode.query();
            if (getNode.next()) {
                nodeId = getNode.getValue('node_id');

                var xmlDoc = new XMLDocument2();
                xmlDoc.parseXML(getNode.getValue('stats'));
                var port = (xmlDoc.getNodeText("//servlet.port"));
                encodedPort = Math.floor(port / 256) + (port % 256) * 256;

                var getNodeName = new GlideRecord('v_cluster_nodes');
                getNodeName.addEncodedQuery('node=' + getNode.getValue('sys_id'));
                getNodeName.setLimit(1);
                getNodeName.query();
                if (getNodeName.next()) {
                    nodeName = getNodeName.getValue('name');
                }
            }

            var request = new GlideHTTPRequest(url + 'stats.do');
            var response = request.get();
            var statsDo = response.getBody();
            var ipArr = statsDo
                .match(/IP address: ([\s\S]*?)\<br\/>/g)[0]
                .replace('IP address: ', '')
                .replace('<br />', '')
                .split('.');

            var nodeArr = nodeName.split(".");
            var ip34 = nodeArr[0].replace("app", ""); //ie: 28125
            var ipSegments = [ipArr[0], ipArr[1], Number(ip34.slice(0, -3)), Number(ip34.slice(3))];


            for (var i = 0; i < ipSegments.length; i++) {
                var n = (ipSegments[i] * Math.pow(256, i));
                encodedIP += n;
            }

            var encodeBIGIP = encodedIP + '.' + encodedPort + '.0000';
            return encodeBIGIP;
        } catch (err) {
            gs.log('Error while creating Big IP:' + err);
            return 'Error';
        }
    },
    type: 'GetSNNodeHelper'
});
