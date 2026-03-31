import type {SidebarsConfig} from '@docusaurus/plugin-content-docs';

const sidebars: SidebarsConfig = {
    docsSidebar: [
        'intro',
        'installation',
        'concepts',
        {
            type: 'category',
            label: 'Tutorial',
            link: {type: 'doc', id: 'tutorial/index'},
            items: [],
        },
        {
            type: 'category',
            label: 'Layout',
            link: {type: 'doc', id: 'layout'},
            items: [
                'layout/vbox-hbox',
                'layout/border',
                'layout/padding',
                'layout/scrollview',
                'layout/grid',
                'layout/spacers',
                'layout/flexchild',
                'layout/alignchild',
            ],
        },
        {
            type: 'category',
            label: 'Widgets',
            items: [
                'widgets/text',
                'widgets/button',
                'widgets/edittext',
                'widgets/list',
                'widgets/componentlist',
                'widgets/checkbox',
                'widgets/label',
                'widgets/progressbar',
                'widgets/notifications',
                'widgets/statusbar',
                'widgets/dialog',
                'widgets/divider',
            ],
        },
        'themes',
        'focus',
        'custom-components',
        'agents',
        'changelog'
    ],
};

export default sidebars;
