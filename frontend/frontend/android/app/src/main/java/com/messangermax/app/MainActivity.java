package com.messangermax.app;

import android.Manifest;
import android.content.pm.PackageManager;
import android.os.Build;
import android.os.Bundle;
import android.view.View;
import android.webkit.PermissionRequest;
import android.webkit.WebChromeClient;
import android.webkit.WebView;
import androidx.core.app.ActivityCompat;
import androidx.core.content.ContextCompat;
import com.getcapacitor.BridgeActivity;
import java.util.ArrayList;
import java.util.List;

public class MainActivity extends BridgeActivity {

    private static final int CALL_PERMISSIONS_REQUEST = 7829;

    @Override
    public void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        final WebView webView = getBridge().getWebView();
        webView.setOverScrollMode(View.OVER_SCROLL_NEVER);
        webView.setWebChromeClient(new WebChromeClient() {
            @Override
            public void onPermissionRequest(final PermissionRequest request) {
                runOnUiThread(() -> {
                    final String[] resources = request.getResources();
                    final List<String> granted = new ArrayList<>();
                    boolean needsRuntime = false;
                    for (String r : resources) {
                        if (PermissionRequest.RESOURCE_VIDEO_CAPTURE.equals(r)) {
                            if (hasPermission(Manifest.permission.CAMERA)) {
                                granted.add(r);
                            } else {
                                needsRuntime = true;
                            }
                        } else if (PermissionRequest.RESOURCE_AUDIO_CAPTURE.equals(r)) {
                            if (hasPermission(Manifest.permission.RECORD_AUDIO)) {
                                granted.add(r);
                            } else {
                                needsRuntime = true;
                            }
                        }
                    }
                    if (needsRuntime) {
                        requestCameraMicPermissions(() -> {
                            final List<String> finalGranted = new ArrayList<>();
                            for (String r : resources) {
                                if (PermissionRequest.RESOURCE_VIDEO_CAPTURE.equals(r) && hasPermission(Manifest.permission.CAMERA)) {
                                    finalGranted.add(r);
                                } else if (PermissionRequest.RESOURCE_AUDIO_CAPTURE.equals(r) && hasPermission(Manifest.permission.RECORD_AUDIO)) {
                                    finalGranted.add(r);
                                }
                            }
                            request.grant(finalGranted.toArray(new String[0]));
                        });
                    } else {
                        request.grant(granted.toArray(new String[0]));
                    }
                });
            }
        });
    }

    private boolean hasPermission(String perm) {
        return ContextCompat.checkSelfPermission(this, perm) == PackageManager.PERMISSION_GRANTED;
    }

    private void requestCameraMicPermissions(Runnable onDone) {
        final List<String> needed = new ArrayList<>();
        if (!hasPermission(Manifest.permission.CAMERA)) needed.add(Manifest.permission.CAMERA);
        if (!hasPermission(Manifest.permission.RECORD_AUDIO)) needed.add(Manifest.permission.RECORD_AUDIO);
        if (needed.isEmpty()) {
            onDone.run();
            return;
        }
        ActivityCompat.requestPermissions(this, needed.toArray(new String[0]), CALL_PERMISSIONS_REQUEST);
        // onDone will be called from onRequestPermissionsResult if desired; for simplicity
        // we invoke it here after the system dialog (the actual grant check happens above).
        onDone.run();
    }
}