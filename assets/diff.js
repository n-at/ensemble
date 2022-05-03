(() => {

    $(() => {
        $('.diff').each(makeDiff);
    });

    function makeDiff(idx, el) {
        const $el = $(el);

        const beforeHeader = $el.find('.diff-before-header').text();
        const before = $el.find('.diff-before-content').text();

        const afterHeader = $el.find('.diff-after-header').text();
        const after = $el.find('.diff-after-content').text();

        const diff = Diff.createTwoFilesPatch(beforeHeader, afterHeader, before, after);

        const configuration = {
            inputFormat: 'diff',
            matching: 'lines',
            outputFormat: 'side-by-side',
            drawFileList: false,
            showFiles: true,
        };
        new Diff2HtmlUI(el, diff, configuration).draw();
    }

})();
